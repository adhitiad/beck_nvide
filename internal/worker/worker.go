package worker

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

// Worker handles background jobs
type Worker struct {
	db          *pgxpool.Pool
	redisClient *redis.Client
	logger      *zap.Logger
	stopChan    chan struct{}
}

// NewWorker creates a new background worker
func NewWorker(db *pgxpool.Pool, r *redis.Client, logger *zap.Logger) *Worker {
	return &Worker{
		db:          db,
		redisClient: r,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the background worker
func (w *Worker) Start() {
	w.logger.Info("Starting background worker")

	storyTicker := time.NewTicker(1 * time.Hour)
	likeSyncTicker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-storyTicker.C:
				w.cleanupExpiredStories()
			case <-likeSyncTicker.C:
				w.syncLikes()
			case <-w.stopChan:
				storyTicker.Stop()
				likeSyncTicker.Stop()
				return
			}
		}
	}()
}

// Stop gracefully stops the background worker
func (w *Worker) Stop() {
	w.logger.Info("Stopping background worker")
	close(w.stopChan)
}

func (w *Worker) cleanupExpiredStories() {
	w.logger.Info("Cleaning up expired stories")
	// Using hard delete for now as per current schema
	query := `DELETE FROM stories WHERE expires_at <= NOW()`
	res, err := w.db.Exec(context.Background(), query)
	if err != nil {
		w.logger.Error("Failed to cleanup expired stories", zap.Error(err))
		return
	}
	w.logger.Info("Expired stories cleanup completed", zap.Int64("deleted_count", res.RowsAffected()))
}

func (w *Worker) syncLikes() {
	w.logger.Info("Syncing likes from cache to DB")
	if w.redisClient == nil {
		return
	}

	ctx := context.Background()
	// Scan for all likes counters
	// Pattern: likes:count:contentType:contentID
	iter := w.redisClient.GetClient().Scan(ctx, 0, "likes:count:*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		parts := strings.Split(key, ":")
		if len(parts) != 4 {
			continue
		}
		contentType := parts[2]
		contentIDStr := parts[3]

		val, err := w.redisClient.GetClient().Get(ctx, key).Result()
		if err != nil {
			continue
		}

		count, err := strconv.Atoi(val)
		if err != nil {
			continue
		}

		contentID, err := domain.FromString(contentIDStr)
		if err != nil {
			continue
		}

		// Update to DB based on content type
		// For simplicity, we just trigger DB queries here.
		// A cleaner approach is to use the repository, but worker might have direct DB access for efficiency.
		switch contentType {
		case "stream", "vod", "story":
			// We only keep likes table for these. But wait, likes table records individual likes.
			// Currently, likes table does NOT have a like_count column, but comments do.
			// Let's check comment like sync
			continue
		case "comment":
			query := `UPDATE comments SET like_count = $1 WHERE id = $2`
			_, err = w.db.Exec(ctx, query, count, contentID)
			if err != nil {
				w.logger.Error("Failed to sync comment like count", zap.Error(err), zap.String("comment_id", contentID.String()))
			}
		}
	}

	if err := iter.Err(); err != nil {
		w.logger.Error("Error scanning redis keys", zap.Error(err))
	}
}

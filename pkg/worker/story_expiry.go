package worker

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/cache"
	"nvide-live/pkg/uuid"
)

// StoryExpiryWorker handles periodic cleanup of expired stories using Redis ZSET scheduling
type StoryExpiryWorker struct {
	cache      *cache.RedisCache
	storyRepo  domain.StoryRepository
	pool       *WorkerPool
	logger     *zap.Logger
	interval   time.Duration
	stopChan   chan struct{}
}

// NewStoryExpiryWorker creates a new StoryExpiryWorker instance
func NewStoryExpiryWorker(
	cache *cache.RedisCache,
	storyRepo domain.StoryRepository,
	pool *WorkerPool,
	logger *zap.Logger,
	interval time.Duration,
) *StoryExpiryWorker {
	return &StoryExpiryWorker{
		cache:     cache,
		storyRepo: storyRepo,
		pool:      pool,
		logger:    logger,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

// Start runs the expiration check loop in a background goroutine
func (w *StoryExpiryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	w.logger.Info("Story expiry worker started", zap.Duration("interval", w.interval))

	go func() {
		for {
			select {
			case <-ticker.C:
				w.logger.Debug("Checking for expired stories in Redis ZSET...")
				w.CheckExpiredStories(ctx)
			case <-w.stopChan:
				ticker.Stop()
				w.logger.Info("Story expiry worker stopped")
				return
			case <-ctx.Done():
				ticker.Stop()
				w.logger.Info("Story expiry worker context cancelled")
				return
			}
		}
	}()
}

// Stop stops the background worker
func (w *StoryExpiryWorker) Stop() {
	close(w.stopChan)
}

// CheckExpiredStories checks and cleans up expired stories
func (w *StoryExpiryWorker) CheckExpiredStories(ctx context.Context) {
	now := time.Now()
	expiredIDs, err := w.cache.GetExpiredStories(ctx, now)
	if err != nil {
		w.logger.Error("Failed to fetch expired stories from Redis ZSET", zap.Error(err))
		return
	}

	if len(expiredIDs) == 0 {
		return
	}

	w.logger.Info("Found expired stories to clean up", zap.Int("count", len(expiredIDs)))

	for _, storyIDStr := range expiredIDs {
		storyID, err := domain.FromString(storyIDStr)
		if err != nil {
			w.logger.Warn("Invalid story UUID in ZSET queue", zap.String("story_id", storyIDStr), zap.Error(err))
			_ = w.cache.RemoveStoryFromExpiryQueue(ctx, storyIDStr)
			continue
		}

		// 1. Fetch story details from DB to get media URL for async deletion
		story, err := w.storyRepo.GetByID(ctx, storyID)
		if err != nil {
			w.logger.Warn("Story not found in DB, removing from Redis queue", zap.String("story_id", storyIDStr), zap.Error(err))
			_ = w.cache.RemoveStoryFromExpiryQueue(ctx, storyIDStr)
			continue
		}

		// 2. Soft delete the story in the database by setting is_expired = true
		story.IsExpired = true
		if err := w.storyRepo.Update(ctx, story); err != nil {
			w.logger.Error("Failed to soft-delete expired story in DB", zap.String("story_id", storyIDStr), zap.Error(err))
			continue
		}

		// 3. Hapus file media secara asinkron lewat Worker Pool
		if story.MediaURL != "" {
			mediaPayload := map[string]string{
				"media_url": story.MediaURL,
				"story_id":  string(story.ID),
			}
			payloadBytes, err := json.Marshal(mediaPayload)
			if err == nil {
				job := &Job{
					ID:         uuid.New(),
					Type:       "media_delete",
					Payload:    payloadBytes,
					MaxRetries: 3,
					CreatedAt:  time.Now(),
				}
				if err := w.pool.Submit(job); err != nil {
					w.logger.Warn("Failed to submit media deletion job to worker pool", zap.String("story_id", storyIDStr), zap.Error(err))
				} else {
					w.logger.Info("Submitted media deletion job asynchronously", zap.String("story_id", storyIDStr), zap.String("media_url", story.MediaURL))
				}
			}
		}

		// 4. Remove story from Redis expiration queue
		if err := w.cache.RemoveStoryFromExpiryQueue(ctx, storyIDStr); err != nil {
			w.logger.Warn("Failed to remove story from expiry queue in Redis", zap.String("story_id", storyIDStr), zap.Error(err))
		}
	}
}

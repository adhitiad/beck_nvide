package worker

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/cache"
	"nvide-live/pkg/uuid"
)

// SocialSyncer handles periodic synchronization of social metrics from Redis to PostgreSQL
type SocialSyncer struct {
	cache      *cache.RedisCache
	storyRepo  domain.StoryRepository
	likeRepo   domain.LikeRepository
	logger     *zap.Logger
	interval   time.Duration
	stopChan   chan struct{}
}

// NewSocialSyncer creates a new SocialSyncer instance
func NewSocialSyncer(
	cache *cache.RedisCache,
	storyRepo domain.StoryRepository,
	likeRepo domain.LikeRepository,
	logger *zap.Logger,
	interval time.Duration,
) *SocialSyncer {
	return &SocialSyncer{
		cache:     cache,
		storyRepo: storyRepo,
		likeRepo:  likeRepo,
		logger:    logger,
		interval:  interval,
		stopChan:  make(chan struct{}),
	}
}

// Start runs the periodic sync loop in a background goroutine
func (s *SocialSyncer) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	s.logger.Info("Social syncer started", zap.Duration("interval", s.interval))

	go func() {
		for {
			select {
			case <-ticker.C:
				s.logger.Info("Starting periodic social cache sync...")
				s.Sync(ctx)
			case <-s.stopChan:
				ticker.Stop()
				s.logger.Info("Social syncer stopped")
				return
			case <-ctx.Done():
				ticker.Stop()
				s.logger.Info("Social syncer context cancelled")
				return
			}
		}
	}()
}

// Stop stops the periodic sync loop
func (s *SocialSyncer) Stop() {
	close(s.stopChan)
}

// Sync performs a single sync operation for story views and likes
func (s *SocialSyncer) Sync(ctx context.Context) {
	// 1. Sync Story Views
	s.syncStoryViews(ctx)

	// 2. Sync Likes
	s.syncLikes(ctx)
}

func (s *SocialSyncer) syncStoryViews(ctx context.Context) {
	storyIDs, err := s.cache.GetStoriesToSyncViews(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch story IDs for view sync", zap.Error(err))
		return
	}

	if len(storyIDs) == 0 {
		return
	}

	s.logger.Info("Syncing story views", zap.Int("count", len(storyIDs)))

	for _, storyIDStr := range storyIDs {
		// Parse story UUID
		storyID, err := domain.FromString(storyIDStr)
		if err != nil {
			s.logger.Warn("Invalid story UUID in sync queue", zap.String("story_id", storyIDStr), zap.Error(err))
			_ = s.cache.RemoveStoryFromSyncViews(ctx, storyIDStr)
			continue
		}

		// Atomically get the cached views and reset count to 0 in Redis
		delta, err := s.cache.GetAndResetStoryViews(ctx, storyIDStr)
		if err != nil {
			s.logger.Error("Failed to get and reset story views", zap.String("story_id", storyIDStr), zap.Error(err))
			continue
		}

		if delta > 0 {
			// Increment view count in PostgreSQL directly with the delta!
			if err := s.storyRepo.AddViewCount(ctx, storyID, int(delta)); err != nil {
				s.logger.Error("Failed to add view count to PostgreSQL", zap.String("story_id", storyIDStr), zap.Int64("delta", delta), zap.Error(err))
				// Note: we can restore the views to Redis if db write fails, but resetting/retrying is standard
				continue
			}
		}

		// Successfully synced, remove from sync queue in Redis
		if err := s.cache.RemoveStoryFromSyncViews(ctx, storyIDStr); err != nil {
			s.logger.Warn("Failed to remove story from sync views queue", zap.String("story_id", storyIDStr), zap.Error(err))
		}
	}
}

func (s *SocialSyncer) syncLikes(ctx context.Context) {
	syncQueue, err := s.cache.GetLikesToSync(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch like sync entries", zap.Error(err))
		return
	}

	if len(syncQueue) == 0 {
		return
	}

	s.logger.Info("Syncing likes batch to database", zap.Int("count", len(syncQueue)))

	for _, entry := range syncQueue {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			s.logger.Warn("Malformed like sync entry", zap.String("entry", entry))
			_ = s.cache.RemoveLikeFromSyncQueue(ctx, entry)
			continue
		}

		contentType := parts[0]
		contentIDStr := parts[1]

		contentID, err := domain.FromString(contentIDStr)
		if err != nil {
			s.logger.Warn("Invalid content UUID in like sync", zap.String("content_id", contentIDStr), zap.Error(err))
			_ = s.cache.RemoveLikeFromSyncQueue(ctx, entry)
			continue
		}

		// Retrieve all user IDs who liked this content from Redis Hash keys
		userIDs, err := s.cache.GetLikeUserIDs(ctx, contentType, contentIDStr)
		if err != nil {
			s.logger.Error("Failed to fetch user likes from Redis hash", zap.String("entry", entry), zap.Error(err))
			continue
		}

		// Create slice of Like structs
		likesList := make([]*domain.Like, 0, len(userIDs))
		for _, uidStr := range userIDs {
			userID, err := domain.FromString(uidStr)
			if err != nil {
				continue
			}

			likesList = append(likesList, &domain.Like{
				ID:          domain.UUID(uuid.New()),
				UserID:      userID,
				ContentID:   contentID,
				ContentType: contentType,
			})
		}

		// Batch sync into PostgreSQL
		if len(likesList) > 0 {
			if err := s.likeRepo.BatchCreate(ctx, likesList); err != nil {
				s.logger.Error("Failed to sync likes batch to DB", zap.String("entry", entry), zap.Error(err))
				continue
			}
		}

		// Successfully synced, remove from queue
		if err := s.cache.RemoveLikeFromSyncQueue(ctx, entry); err != nil {
			s.logger.Warn("Failed to remove like entry from sync queue", zap.String("entry", entry), zap.Error(err))
		}
	}
}

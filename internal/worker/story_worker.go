package worker

import (
	"context"
	"time"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

// StoryWorker handles background tasks for stories
type StoryWorker struct {
	storyRepo domain.StoryRepository
	logger    *zap.Logger
}

// NewStoryWorker creates a new story worker
func NewStoryWorker(storyRepo domain.StoryRepository, logger *zap.Logger) *StoryWorker {
	return &StoryWorker{
		storyRepo: storyRepo,
		logger:    logger,
	}
}

// Start starts the background worker
func (w *StoryWorker) Start(ctx context.Context) {
	w.logger.Info("Story cleanup worker started")
	
	// Run immediately on start
	w.cleanup(ctx)

	ticker := time.NewTicker(24 * time.Hour) // Run once a day
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Story cleanup worker stopping")
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *StoryWorker) cleanup(ctx context.Context) {
	w.logger.Info("Running story cleanup...")
	err := w.storyRepo.DeleteExpired(ctx)
	if err != nil {
		w.logger.Error("Failed to cleanup expired stories", zap.Error(err))
	} else {
		w.logger.Info("Story cleanup completed successfully")
	}
}

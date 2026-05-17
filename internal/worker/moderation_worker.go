package worker

import (
	"context"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type ModerationWorker struct {
	moderationUC domain.ModerationUseCase
	logger       *zap.Logger
	stopChan     chan struct{}
}

func NewModerationWorker(moderationUC domain.ModerationUseCase, logger *zap.Logger) *ModerationWorker {
	return &ModerationWorker{
		moderationUC: moderationUC,
		logger:       logger,
		stopChan:     make(chan struct{}),
	}
}

func (w *ModerationWorker) Start(ctx context.Context) {
	w.logger.Info("Moderation & Safety Engine workers starting...")

	imageTicker := time.NewTicker(5 * time.Second)
	syncTicker := time.NewTicker(60 * time.Second)

	defer imageTicker.Stop()
	defer syncTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Moderation worker context cancelled, stopping...")
			return
		case <-w.stopChan:
			w.logger.Info("Moderation worker manually stopped.")
			return
		case <-imageTicker.C:
			w.processPendingImages(ctx)
		case <-syncTicker.C:
			w.syncBanMuteStates(ctx)
		}
	}
}

func (w *ModerationWorker) Stop() {
	close(w.stopChan)
}

func (w *ModerationWorker) processPendingImages(ctx context.Context) {
	// Execute background image scan
	err := w.moderationUC.ProcessNextImageInQueue(ctx)
	if err != nil {
		w.logger.Debug("Queue empty or failed to process pending image job", zap.Error(err))
	}
}

func (w *ModerationWorker) syncBanMuteStates(ctx context.Context) {
	// Reconcile and lift muted/banned users when TTL expires
	err := w.moderationUC.SyncUserModerationStates(ctx)
	if err != nil {
		w.logger.Error("Failed to sync expired moderation states", zap.Error(err))
	}
}

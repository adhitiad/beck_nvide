package worker

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
)

type ReplicationWorker struct {
	strategy *usecase.StorageStrategy
	repo     domain.StorageFileRepository
	logger   *zap.Logger
	interval time.Duration
	stopChan chan struct{}
}

type ReplicationWorkerConfig struct {
	Strategy *usecase.StorageStrategy
	Repo     domain.StorageFileRepository
	Logger   *zap.Logger
	Interval time.Duration
}

func NewReplicationWorker(cfg *ReplicationWorkerConfig) *ReplicationWorker {
	if cfg.Interval == 0 {
		cfg.Interval = 5 * time.Minute
	}
	return &ReplicationWorker{
		strategy: cfg.Strategy,
		repo:     cfg.Repo,
		logger:   cfg.Logger,
		interval: cfg.Interval,
		stopChan: make(chan struct{}),
	}
}

func (w *ReplicationWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	w.logger.Info("Replication worker started", zap.Duration("interval", w.interval))

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopChan:
				return
			case <-ticker.C:
				w.processReplications(ctx)
			}
		}
	}()
}

func (w *ReplicationWorker) Stop() {
	close(w.stopChan)
}

func (w *ReplicationWorker) processReplications(ctx context.Context) {
	files, err := w.repo.ListUnreplicated(ctx, 100)
	if err != nil {
		w.logger.Error("Failed to list unreplicated files", zap.Error(err))
		return
	}

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return
		}
		w.replicateFile(ctx, file)
	}
}

func (w *ReplicationWorker) replicateFile(ctx context.Context, file *domain.StorageFile) {
	contentType := domain.ContentType(file.ContentType)
	_, expectedReplicas, err := w.strategy.GetPrimaryProvider(contentType)
	if err != nil {
		w.logger.Error("Failed to get expected replicas", zap.Error(err))
		return
	}

	needReplicas := make([]domain.StorageProvider, 0)
	for _, expProvider := range expectedReplicas {
		found := false
		for _, replicatedTo := range file.ReplicatedTo {
			if replicatedTo == expProvider.Name() {
				found = true
				break
			}
		}
		if !found {
			needReplicas = append(needReplicas, expProvider)
		}
	}

	if len(needReplicas) == 0 {
		return
	}

	for _, provider := range needReplicas {
		select {
		case <-ctx.Done():
			return
		default:
			reader, err := w.createReader(ctx, file)
			if err != nil {
				w.logger.Error("Failed to create reader for replication", zap.Error(err))
				continue
			}

			_, err = provider.Upload(ctx, file.Bucket, file.Key, reader, file.ContentType, file.Metadata)
			if err != nil {
				w.logger.Error("Replication to provider failed",
					zap.String("provider", provider.Name()),
					zap.String("key", file.Key),
					zap.Error(err))
				w.sendAlert(file, provider.Name(), err)
				continue
			}

			file.ReplicatedTo = append(file.ReplicatedTo, provider.Name())
			w.logger.Info("Replication completed",
				zap.String("provider", provider.Name()),
				zap.String("key", file.Key))
		}
	}
}

func (w *ReplicationWorker) createReader(ctx context.Context, file *domain.StorageFile) (io.Reader, error) {
	return nil, nil
}

func (w *ReplicationWorker) sendAlert(file *domain.StorageFile, providerName string, err error) {
	w.logger.Error("Replication failed - alert",
		zap.String("file_id", file.ID.String()),
		zap.String("provider", providerName),
		zap.Error(err))
}
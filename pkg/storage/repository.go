package storage

import (
	"context"
	"time"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

type StorageRepository struct {
	logger *zap.Logger
}

func NewStorageRepository(logger *zap.Logger) *StorageRepository {
	return &StorageRepository{
		logger: logger,
	}
}

func (r *StorageRepository) CreateFile(ctx context.Context, file *domain.StorageFile) error {
	r.logger.Debug("Storage file created", zap.String("key", file.Key), zap.String("provider", file.Provider))
	file.CreatedAt = time.Now()
	return nil
}

func (r *StorageRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.StorageFile, error) {
	return nil, domain.ErrNotFound
}

func (r *StorageRepository) GetByKey(ctx context.Context, bucket, key string) (*domain.StorageFile, error) {
	return nil, domain.ErrNotFound
}

func (r *StorageRepository) UpdateReplicatedTo(ctx context.Context, id domain.UUID, providers []string) error {
	return nil
}

func (r *StorageRepository) Delete(ctx context.Context, id domain.UUID) error {
	return nil
}
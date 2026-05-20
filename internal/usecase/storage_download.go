package usecase

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type StorageDownloadUseCase struct {
	strategy *StorageStrategy
	repo     domain.StorageFileRepository
	logger   *zap.Logger
}

type StorageDownloadConfig struct {
	Strategy *StorageStrategy
	Repo     domain.StorageFileRepository
	Logger   *zap.Logger
}

func NewStorageDownloadUseCase(cfg *StorageDownloadConfig) *StorageDownloadUseCase {
	return &StorageDownloadUseCase{
		strategy: cfg.Strategy,
		repo:     cfg.Repo,
		logger:   cfg.Logger,
	}
}

func (uc *StorageDownloadUseCase) GetPresignedURL(ctx context.Context, fileID domain.UUID) (string, error) {
	return uc.GetPresignedURLWithExpiry(ctx, fileID, 5*time.Minute)
}

func (uc *StorageDownloadUseCase) GetPresignedURLWithExpiry(ctx context.Context, fileID domain.UUID, expiry time.Duration) (string, error) {
	storageFile, err := uc.repo.GetByID(ctx, fileID)
	if err != nil {
		return "", err
	}

	provider := uc.strategy.GetProvider(domain.StorageProviderType(storageFile.Provider))
	if provider == nil {
		return "", domain.NewDomainError(domain.ErrCodeNotFound, "primary provider not found", nil)
	}

	url, err := provider.GetPresignedURL(ctx, storageFile.Bucket, storageFile.Key, expiry)
	if err != nil {
		uc.logger.Error("Primary provider presigned URL failed, trying replicas", zap.Error(err))
		return uc.getReplicaURL(ctx, storageFile, expiry)
	}

	return url, nil
}

func (uc *StorageDownloadUseCase) GetPresignedURLs(ctx context.Context, fileID domain.UUID) (map[string]string, error) {
	return uc.GetPresignedURLsWithExpiry(ctx, fileID, 5*time.Minute)
}

func (uc *StorageDownloadUseCase) GetPresignedURLsWithExpiry(ctx context.Context, fileID domain.UUID, expiry time.Duration) (map[string]string, error) {
	storageFile, err := uc.repo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	urls := make(map[string]string)

	provider := uc.strategy.GetProvider(domain.StorageProviderType(storageFile.Provider))
	if provider == nil {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "primary provider not found", nil)
	}

	url, err := provider.GetPresignedURL(ctx, storageFile.Bucket, storageFile.Key, expiry)
	if err != nil {
		uc.logger.Warn("Primary provider failed", zap.Error(err))
	} else {
		urls[storageFile.Provider] = url
	}

	for _, providerName := range storageFile.ReplicatedTo {
		p := uc.strategy.GetProvider(domain.StorageProviderType(providerName))
		if p != nil {
			url, err := p.GetPresignedURL(ctx, storageFile.Bucket, storageFile.Key, expiry)
			if err != nil {
				uc.logger.Warn("Replica provider failed", zap.String("provider", providerName), zap.Error(err))
				continue
			}
			urls[providerName] = url
		}
	}

	if len(urls) == 0 {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "no available provider for file", nil)
	}

	return urls, nil
}

func (uc *StorageDownloadUseCase) Download(ctx context.Context, bucket, key string, expectedProvider string) (io.ReadCloser, error) {
	var provider domain.StorageProvider
	if expectedProvider != "" {
		provider = uc.strategy.GetProvider(domain.StorageProviderType(expectedProvider))
	}

	if provider == nil {
		storageFile, err := uc.repo.GetByKey(ctx, bucket, key)
		if err != nil {
			return nil, err
		}
		provider = uc.strategy.GetProvider(domain.StorageProviderType(storageFile.Provider))
	}

	if provider == nil {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "provider not found", nil)
	}

	return provider.Download(ctx, bucket, key)
}

func (uc *StorageDownloadUseCase) getReplicaURL(ctx context.Context, storageFile *domain.StorageFile, expiry time.Duration) (string, error) {
	for _, providerName := range storageFile.ReplicatedTo {
		provider := uc.strategy.GetProvider(domain.StorageProviderType(providerName))
		if provider == nil {
			continue
		}

		url, err := provider.GetPresignedURL(ctx, storageFile.Bucket, storageFile.Key, expiry)
		if err != nil {
			uc.logger.Error("Replica provider presigned URL failed", zap.String("provider", providerName), zap.Error(err))
			continue
		}

		return url, nil
	}

	return "", domain.NewDomainError(domain.ErrCodeInternal, "all providers failed to generate presigned URL", nil)
}
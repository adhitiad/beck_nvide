package storage

import (
	"context"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

type StrategyInterface interface {
	GetPrimaryProvider(contentType domain.ContentType) (domain.StorageProvider, error)
	GetReplicationProviders(contentType domain.ContentType) ([]domain.StorageProvider, error)
}

type MultiStorage struct {
	providers    map[domain.StorageProviderType]domain.StorageProvider
	strategy     StrategyInterface
	logger       *zap.Logger
	mu           sync.RWMutex
}

type MultiStorageConfig struct {
	Providers map[domain.StorageProviderType]domain.StorageProvider
	Strategy  StrategyInterface
	Logger    *zap.Logger
}

func NewMultiStorage(cfg *MultiStorageConfig) *MultiStorage {
	return &MultiStorage{
		providers: cfg.Providers,
		strategy:  cfg.Strategy,
		logger:    cfg.Logger,
	}
}

func (s *MultiStorage) Upload(ctx context.Context, key string, body io.Reader, contentType domain.ContentType, metadata map[string]string) (*domain.UploadResult, error) {
	primaryProvider, err := s.strategy.GetPrimaryProvider(contentType)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	bucket := s.getBucketForContentType(contentType, primaryProvider.Name())
	s.mu.RUnlock()

	result, err := primaryProvider.Upload(ctx, bucket, key, body, string(contentType), metadata)
	if err != nil {
		return nil, err
	}

	go s.replicateAsync(ctx, primaryProvider, result, contentType)

	return result, nil
}

func (s *MultiStorage) Download(ctx context.Context, providerName, bucket, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	provider, exists := s.providers[domain.StorageProviderType(providerName)]
	s.mu.RUnlock()

	if !exists {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "provider not found: "+providerName, nil)
	}

	return provider.Download(ctx, bucket, key)
}

func (s *MultiStorage) Delete(ctx context.Context, key string, contentType domain.ContentType) error {
	s.mu.RLock()
	primaryProvider, err := s.strategy.GetPrimaryProvider(contentType)
	if err != nil {
		s.mu.RUnlock()
		return err
	}
	bucket := s.getBucketForContentType(contentType, primaryProvider.Name())
	replicationProviders, _ := s.strategy.GetReplicationProviders(contentType)
	s.mu.RUnlock()

	if err := primaryProvider.Delete(ctx, bucket, key); err != nil {
		s.logger.Warn("Failed to delete from primary provider", zap.Error(err))
	}

	for _, rp := range replicationProviders {
		if err := rp.Delete(ctx, bucket, key); err != nil {
			s.logger.Warn("Failed to delete from replication provider", zap.String("provider", rp.Name()), zap.Error(err))
		}
	}

	return nil
}

func (s *MultiStorage) GetPresignedURL(ctx context.Context, providerName, bucket, key string, expiry time.Duration) (string, error) {
	s.mu.RLock()
	provider, exists := s.providers[domain.StorageProviderType(providerName)]
	s.mu.RUnlock()

	if !exists {
		return "", domain.NewDomainError(domain.ErrCodeNotFound, "provider not found: "+providerName, nil)
	}

	return provider.GetPresignedURL(ctx, bucket, key, expiry)
}

func (s *MultiStorage) replicateAsync(ctx context.Context, primaryProvider domain.StorageProvider, result *domain.UploadResult, contentType domain.ContentType) {
	replicationProviders, err := s.strategy.GetReplicationProviders(contentType)
	if err != nil {
		s.logger.Error("Failed to get replication providers", zap.Error(err))
		return
	}

	for _, rp := range replicationProviders {
		go func(targetProvider domain.StorageProvider) {
			s.logger.Info("Starting async replication",
				zap.String("source", primaryProvider.Name()),
				zap.String("target", targetProvider.Name()),
				zap.String("key", result.Key))

			reader, err := primaryProvider.Download(ctx, result.Bucket, result.Key)
			if err != nil {
				s.logger.Error("Failed to download from primary for replication", zap.Error(err))
				return
			}

			_, err = targetProvider.Upload(ctx, result.Bucket, result.Key, reader, "", result.Metadata)
			reader.Close()
			if err != nil {
				s.logger.Error("Replication failed", zap.String("target", targetProvider.Name()), zap.Error(err))
				return
			}

			s.logger.Info("Replication completed successfully",
				zap.String("source", primaryProvider.Name()),
				zap.String("target", targetProvider.Name()))
		}(rp)
	}
}

func (s *MultiStorage) getBucketForContentType(contentType domain.ContentType, providerName string) string {
	switch domain.StorageProviderType(providerName) {
	case domain.ProviderOCI:
		switch contentType {
		case domain.ContentTypeVideo, domain.ContentTypeVOD:
			return "nvide-videos"
		case domain.ContentTypeThumbnail:
			return "nvide-thumbnails"
		default:
			return "nvide-storage"
		}
	case domain.ProviderStorj:
		return "nvide-private"
	case domain.ProviderFilebase:
		return "nvide-backup"
	default:
		return "nvide-storage"
	}
}

func (s *MultiStorage) RegisterProvider(providerType domain.StorageProviderType, provider domain.StorageProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[providerType] = provider
}

func (s *MultiStorage) GetProvider(providerType domain.StorageProviderType) (domain.StorageProvider, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, exists := s.providers[providerType]
	return p, exists
}

func (s *MultiStorage) GetStrategy() StrategyInterface {
	return s.strategy
}
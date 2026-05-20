package storage

import (
	"nvide-live/internal/domain"
	"sync"
)

type ContentStorageStrategy struct {
	providers map[domain.StorageProviderType]domain.StorageProvider
	mu        sync.RWMutex
}

type ContentStrategyConfig struct {
	Providers map[domain.StorageProviderType]domain.StorageProvider
}

func NewContentStorageStrategy(cfg *ContentStrategyConfig) *ContentStorageStrategy {
	return &ContentStorageStrategy{
		providers: cfg.Providers,
	}
}

func (s *ContentStorageStrategy) GetPrimaryProvider(contentType domain.ContentType) (domain.StorageProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch contentType {
	case domain.ContentTypeVideo, domain.ContentTypeVOD:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			return p, nil
		}
	case domain.ContentTypeThumbnail:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			return p, nil
		}
	case domain.ContentTypePrivateCall:
		if p, ok := s.providers[domain.ProviderStorj]; ok {
			return p, nil
		}
	case domain.ContentTypeBackup:
		if p, ok := s.providers[domain.ProviderFilebase]; ok {
			return p, nil
		}
	case domain.ContentTypeClip:
		if p, ok := s.providers[domain.ProviderFilebase]; ok {
			return p, nil
		}
	}

	if p, ok := s.providers[domain.ProviderOCI]; ok {
		return p, nil
	}

	return nil, domain.NewDomainError(domain.ErrCodeInternal, "no storage provider available", nil)
}

func (s *ContentStorageStrategy) GetReplicationProviders(contentType domain.ContentType) ([]domain.StorageProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []domain.StorageProvider

	switch contentType {
	case domain.ContentTypeVideo, domain.ContentTypeVOD, domain.ContentTypeThumbnail:
		if p, ok := s.providers[domain.ProviderStorj]; ok {
			result = append(result, p)
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok {
			result = append(result, p)
		}
	case domain.ContentTypePrivateCall:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			result = append(result, p)
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok {
			result = append(result, p)
		}
	case domain.ContentTypeClip:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			result = append(result, p)
		}
		if p, ok := s.providers[domain.ProviderStorj]; ok {
			result = append(result, p)
		}
	default:
		if p, ok := s.providers[domain.ProviderFilebase]; ok {
			result = append(result, p)
		}
		if p, ok := s.providers[domain.ProviderStorj]; ok {
			result = append(result, p)
		}
	}

	return result, nil
}

func (s *ContentStorageStrategy) RegisterProvider(providerType domain.StorageProviderType, provider domain.StorageProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[providerType] = provider
}
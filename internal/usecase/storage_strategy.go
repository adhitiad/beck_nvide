package usecase

import (
	"nvide-live/internal/domain"
	"sync"
)

type StorageStrategy struct {
	providers map[domain.StorageProviderType]domain.StorageProvider
	mu        sync.RWMutex
}

type StorageStrategyConfig struct {
	Providers map[domain.StorageProviderType]domain.StorageProvider
}

func NewStorageStrategy(cfg *StorageStrategyConfig) *StorageStrategy {
	return &StorageStrategy{
		providers: cfg.Providers,
	}
}

func (s *StorageStrategy) GetPrimaryProvider(contentType domain.ContentType) (domain.StorageProvider, []domain.StorageProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var primary domain.StorageProvider
	var replicators []domain.StorageProvider

	switch contentType {
	case domain.ContentTypeVideo:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			primary = p
		}
		if p, ok := s.providers[domain.ProviderStorj]; ok && primary != p {
			replicators = append(replicators, p)
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok && primary != p {
			replicators = append(replicators, p)
		}
	case domain.ContentTypeVOD:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			primary = p
		}
		if p, ok := s.providers[domain.ProviderStorj]; ok && primary != p {
			replicators = append(replicators, p)
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok && primary != p {
			replicators = append(replicators, p)
		}
	case domain.ContentTypeVODPremium:
		if p, ok := s.providers[domain.ProviderStorj]; ok {
			primary = p
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok && primary != p {
			replicators = append(replicators, p)
		}
	case domain.ContentTypeThumbnail:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			primary = p
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok && primary != p {
			replicators = append(replicators, p)
		}
	case domain.ContentTypeClip:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			primary = p
		}
		if p, ok := s.providers[domain.ProviderFilebase]; ok && primary != p {
			replicators = append(replicators, p)
		}
	case domain.ContentTypePrivateCall:
		if p, ok := s.providers[domain.ProviderStorj]; ok {
			primary = p
		}
		if p, ok := s.providers[domain.ProviderOCI]; ok && primary != p {
			replicators = append(replicators, p)
		}
	case domain.ContentTypeKYC:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			primary = p
		}
	default:
		if p, ok := s.providers[domain.ProviderOCI]; ok {
			primary = p
		}
	}

	if primary == nil {
		return nil, nil, domain.NewDomainError(domain.ErrCodeInternal, "no primary storage provider available", nil)
	}

	return primary, replicators, nil
}

func (s *StorageStrategy) RegisterProvider(providerType domain.StorageProviderType, provider domain.StorageProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[providerType] = provider
}

func (s *StorageStrategy) GetProvider(providerType domain.StorageProviderType) domain.StorageProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.providers[providerType]
}
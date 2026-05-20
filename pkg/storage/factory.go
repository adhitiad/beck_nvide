package storage

import (
	"time"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

type StorageFactory struct {
	logger *zap.Logger
}

func NewStorageFactory(logger *zap.Logger) *StorageFactory {
	return &StorageFactory{
		logger: logger,
	}
}

func (f *StorageFactory) CreateOCIProvider(cfg *OCIConfig) *OCIProvider {
	return NewOCIProvider(cfg, f.logger)
}

func (f *StorageFactory) CreateStorjProvider(cfg *StorjConfig) *StorjProvider {
	return NewStorjProvider(cfg, f.logger)
}

func (f *StorageFactory) CreateFilebaseProvider(cfg *FilebaseConfig) *FilebaseProvider {
	return NewFilebaseProvider(cfg, f.logger)
}

func (f *StorageFactory) CreateMultiStorage(providers map[domain.StorageProviderType]domain.StorageProvider) *MultiStorage {
	strategy := NewContentStorageStrategy(&ContentStrategyConfig{
		Providers: providers,
	})

	return NewMultiStorage(&MultiStorageConfig{
		Providers: providers,
		Strategy:  strategy,
		Logger:    f.logger,
	})
}

func (f *StorageFactory) CreateMultiStorageWithStrategy(providers map[domain.StorageProviderType]domain.StorageProvider, strategy StrategyInterface) *MultiStorage {
	return NewMultiStorage(&MultiStorageConfig{
		Providers: providers,
		Strategy:  strategy,
		Logger:    f.logger,
	})
}

func (f *StorageFactory) CreateDefaultMultiStorage(ociCfg *OCIConfig, storjCfg *StorjConfig, filebaseCfg *FilebaseConfig) *MultiStorage {
	ociProvider := f.CreateOCIProvider(ociCfg)
	storjProvider := f.CreateStorjProvider(storjCfg)
	filebaseProvider := f.CreateFilebaseProvider(filebaseCfg)

	providers := map[domain.StorageProviderType]domain.StorageProvider{
		domain.ProviderOCI:     ociProvider,
		domain.ProviderStorj:   storjProvider,
		domain.ProviderFilebase: filebaseProvider,
	}

	return f.CreateMultiStorage(providers)
}

type StorageOptions struct {
	EnableReplication bool
	ReplicationDelay  time.Duration
	MaxRetryAttempts  int
}

func DefaultStorageOptions() *StorageOptions {
	return &StorageOptions{
		EnableReplication: true,
		ReplicationDelay:  5 * time.Second,
		MaxRetryAttempts:  3,
	}
}
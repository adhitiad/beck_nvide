package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type StorageUploadUseCase struct {
	strategy *StorageStrategy
	repo     domain.StorageFileRepository
	logger   *zap.Logger
}

type StorageUploadConfig struct {
	Strategy *StorageStrategy
	Repo     domain.StorageFileRepository
	Logger   *zap.Logger
}

func NewStorageUploadUseCase(cfg *StorageUploadConfig) *StorageUploadUseCase {
	return &StorageUploadUseCase{
		strategy: cfg.Strategy,
		repo:     cfg.Repo,
		logger:   cfg.Logger,
	}
}

func (uc *StorageUploadUseCase) GenerateKey() string {
	return fmt.Sprintf("%s_%d", uuid.New().String(), time.Now().UnixNano())
}

func (uc *StorageUploadUseCase) Upload(ctx context.Context, bucket string, key string, body interface {
	Bytes() ([]byte, error)
}, contentType domain.ContentType, metadata map[string]string) (*domain.StorageFile, error) {
	primaryProvider, replicators, err := uc.strategy.GetPrimaryProvider(contentType)
	if err != nil {
		return nil, err
	}

	bytesData, err := body.Bytes()
	if err != nil {
		return nil, err
	}

	uploadResult, err := primaryProvider.Upload(ctx, bucket, key, bytes.NewReader(bytesData), string(contentType), metadata)
	if err != nil {
		return nil, err
	}

	storageFile := &domain.StorageFile{
		ID:           domain.NewUUID(),
		Bucket:       bucket,
		Key:          uploadResult.Key,
		Provider:     uploadResult.Provider,
		Size:         uploadResult.Size,
		ContentType:  string(contentType),
		Metadata:     metadata,
		CreatedAt:    time.Now(),
		ReplicatedTo: []string{},
	}

	if len(replicators) > 0 {
		go uc.replicateAsync(ctx, storageFile, replicators, bucket, key, contentType, metadata, bytes.NewReader(bytesData))
	}

	return storageFile, nil
}

func (uc *StorageUploadUseCase) UploadWithReader(ctx context.Context, bucket string, key string, body io.Reader, contentType domain.ContentType, metadata map[string]string) (*domain.StorageFile, error) {
	return uc.UploadWithReaderAndBytes(ctx, bucket, key, body, nil, contentType, metadata)
}

func (uc *StorageUploadUseCase) UploadWithReaderAndBytes(ctx context.Context, bucket string, key string, body io.Reader, bytesData []byte, contentType domain.ContentType, metadata map[string]string) (*domain.StorageFile, error) {
	primaryProvider, replicators, err := uc.strategy.GetPrimaryProvider(contentType)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if bytesData != nil {
		reader = bytes.NewReader(bytesData)
	} else if body != nil {
		reader = body
	}

	if reader == nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "body is required", nil)
	}

	uploadResult, err := primaryProvider.Upload(ctx, bucket, key, reader, string(contentType), metadata)
	if err != nil {
		return nil, err
	}

	storageFile := &domain.StorageFile{
		ID:           domain.NewUUID(),
		Bucket:       bucket,
		Key:          uploadResult.Key,
		Provider:     uploadResult.Provider,
		Size:         uploadResult.Size,
		ContentType:  string(contentType),
		Metadata:     metadata,
		CreatedAt:    time.Now(),
		ReplicatedTo: []string{},
	}

	if len(replicators) > 0 && bytesData != nil {
		go uc.replicateAsync(ctx, storageFile, replicators, bucket, key, contentType, metadata, bytes.NewReader(bytesData))
	}

	return storageFile, nil
}

func (uc *StorageUploadUseCase) replicateAsync(ctx context.Context, storageFile *domain.StorageFile, replicators []domain.StorageProvider, bucket, key string, contentType domain.ContentType, metadata map[string]string, body io.Reader) {
	for _, provider := range replicators {
		select {
		case <-ctx.Done():
			return
		default:
			reader, ok := body.(*bytes.Reader)
			if ok {
				_, _ = reader.Seek(0, io.SeekStart)
			}
			_, err := provider.Upload(ctx, bucket, key, reader, string(contentType), metadata)
			if err != nil {
				uc.logger.Error("Replication failed", zap.String("provider", provider.Name()), zap.String("key", key), zap.Error(err))
				continue
			}
			storageFile.ReplicatedTo = append(storageFile.ReplicatedTo, provider.Name())
			uc.logger.Info("Replication completed", zap.String("provider", provider.Name()), zap.String("key", key))
		}
	}
}

func (uc *StorageUploadUseCase) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	storageFile, err := uc.repo.GetByKey(ctx, bucket, key)
	if err != nil {
		return nil, err
	}

	provider := uc.strategy.GetProvider(domain.StorageProviderType(storageFile.Provider))
	if provider == nil {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "provider not found", nil)
	}

	return provider.Download(ctx, bucket, key)
}

func (uc *StorageUploadUseCase) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	storageFile, err := uc.repo.GetByKey(ctx, bucket, key)
	if err != nil {
		return "", err
	}

	provider := uc.strategy.GetProvider(domain.StorageProviderType(storageFile.Provider))
	if provider == nil {
		return "", domain.NewDomainError(domain.ErrCodeNotFound, "provider not found", nil)
	}

	return provider.GetPresignedURL(ctx, bucket, key, expiry)
}

func (uc *StorageUploadUseCase) Delete(ctx context.Context, bucket, key string) error {
	storageFile, err := uc.repo.GetByKey(ctx, bucket, key)
	if err != nil {
		return err
	}

	provider := uc.strategy.GetProvider(domain.StorageProviderType(storageFile.Provider))
	if provider == nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "provider not found", nil)
	}

	return provider.Delete(ctx, bucket, key)
}
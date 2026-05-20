package storage

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

type StorjProvider struct {
	client *s3.Client
	logger *zap.Logger
}

type StorjConfig struct {
	AccessKey string
	SecretKey string
	Endpoint  string
}

func NewStorjProvider(cfg *StorjConfig, logger *zap.Logger) *StorjProvider {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service string, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			SigningRegion:     "us-east-1",
			HostnameImmutable: true,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		logger.Fatal("Failed to create Storj config", zap.Error(err))
	}

	return &StorjProvider{
		client: s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = true
		}),
		logger: logger,
	}
}

func (p *StorjProvider) Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string) (*domain.UploadResult, error) {
	p.logger.Debug("Storj Upload", zap.String("bucket", bucket), zap.String("key", key))

	uploader := manager.NewUploader(p.client)

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	})
	if err != nil {
		p.logger.Error("Storj upload failed", zap.Error(err))
		return nil, err
	}

	url := "https://gateway.storjshare.io/" + bucket + "/" + key

	return &domain.UploadResult{
		URL:      url,
		Key:      key,
		Bucket:   bucket,
		Provider: string(domain.ProviderStorj),
		Metadata: metadata,
	}, nil
}

func (p *StorjProvider) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	p.logger.Debug("Storj Download", zap.String("bucket", bucket), zap.String("key", key))

	resp, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		p.logger.Error("Storj download failed", zap.Error(err))
		return nil, err
	}

	return resp.Body, nil
}

func (p *StorjProvider) Delete(ctx context.Context, bucket, key string) error {
	p.logger.Debug("Storj Delete", zap.String("bucket", bucket), zap.String("key", key))

	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		p.logger.Error("Storj delete failed", zap.Error(err))
		return err
	}

	return nil
}

func (p *StorjProvider) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	p.logger.Debug("Storj GetPresignedURL", zap.String("bucket", bucket), zap.String("key", key))

	presigner := s3.NewPresignClient(p.client)

	presignedURL, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		p.logger.Error("Storj presigned URL creation failed", zap.Error(err))
		return "", err
	}

	return presignedURL.URL, nil
}

func (p *StorjProvider) Name() string {
	return string(domain.ProviderStorj)
}

func (p *StorjProvider) UploadMultipart(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string, partSize int64) (*domain.UploadResult, error) {
	uploader := manager.NewUploader(p.client, func(u *manager.Uploader) {
		u.PartSize = partSize
	})

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	})
	if err != nil {
		return nil, err
	}

	url := "https://gateway.storjshare.io/" + bucket + "/" + key
	return &domain.UploadResult{
		URL:      url,
		Key:      key,
		Bucket:   bucket,
		Provider: string(domain.ProviderStorj),
		Metadata: metadata,
	}, nil
}

func LoadStorjConfigFromEnv() *StorjConfig {
	return &StorjConfig{
		AccessKey: os.Getenv("STORJ_ACCESS_KEY"),
		SecretKey: os.Getenv("STORJ_SECRET_KEY"),
		Endpoint:  "https://gateway.storjshare.io",
	}
}
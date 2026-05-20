package repository

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

type FilebaseProvider struct {
	client *s3.Client
	logger *zap.Logger
}

type FilebaseConfig struct {
	AccessKey string
	SecretKey string
	Endpoint  string
}

func NewFilebaseProvider(cfg *FilebaseConfig, logger *zap.Logger) *FilebaseProvider {
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
		logger.Fatal("Failed to create Filebase config", zap.Error(err))
	}

	return &FilebaseProvider{
		client: s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.UsePathStyle = true
		}),
		logger: logger,
	}
}

func (p *FilebaseProvider) Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string) (*domain.UploadResult, error) {
	p.logger.Debug("Filebase Upload", zap.String("bucket", bucket), zap.String("key", key))

	uploader := manager.NewUploader(p.client)

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
		Metadata:    metadata,
	})
	if err != nil {
		p.logger.Error("Filebase upload failed", zap.Error(err))
		return nil, err
	}

	ipfsHash := ""
	if h, err := p.GetIPFSHash(ctx, bucket, key); err == nil {
		ipfsHash = h
	}

	url := "https://" + bucket + ".s3.filebase.com/" + key

	return &domain.UploadResult{
		URL:      url,
		Key:      key,
		Bucket:   bucket,
		Provider: string(domain.ProviderFilebase),
		IPFSHash: ipfsHash,
		Metadata: metadata,
	}, nil
}

func (p *FilebaseProvider) UploadMultipart(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string, partSize int64) (*domain.UploadResult, error) {
	p.logger.Debug("Filebase Multipart Upload", zap.String("bucket", bucket), zap.String("key", key))

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
		p.logger.Error("Filebase multipart upload failed", zap.Error(err))
		return nil, err
	}

	ipfsHash := ""
	if h, err := p.GetIPFSHash(ctx, bucket, key); err == nil {
		ipfsHash = h
	}

	url := "https://" + bucket + ".s3.filebase.com/" + key

	return &domain.UploadResult{
		URL:      url,
		Key:      key,
		Bucket:   bucket,
		Provider: string(domain.ProviderFilebase),
		IPFSHash: ipfsHash,
		Metadata: metadata,
	}, nil
}

func (p *FilebaseProvider) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	p.logger.Debug("Filebase Download", zap.String("bucket", bucket), zap.String("key", key))

	resp, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		p.logger.Error("Filebase download failed", zap.Error(err))
		return nil, err
	}

	return resp.Body, nil
}

func (p *FilebaseProvider) GetIPFSHash(ctx context.Context, bucket, key string) (string, error) {
	resp, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}

	if resp.Metadata != nil {
		if ipfsHash, ok := resp.Metadata["ipfs-hash"]; ok && ipfsHash != "" {
			return ipfsHash, nil
		}
		if ipfsHash, ok := resp.Metadata["x-amz-meta-ipfs-hash"]; ok && ipfsHash != "" {
			return ipfsHash, nil
		}
	}

	return "", nil
}

func (p *FilebaseProvider) Delete(ctx context.Context, bucket, key string) error {
	p.logger.Debug("Filebase Delete", zap.String("bucket", bucket), zap.String("key", key))

	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		p.logger.Error("Filebase delete failed", zap.Error(err))
		return err
	}

	return nil
}

func (p *FilebaseProvider) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	p.logger.Debug("Filebase GetPresignedURL", zap.String("bucket", bucket), zap.String("key", key))

	presigner := s3.NewPresignClient(p.client)

	presignedURL, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		p.logger.Error("Filebase presigned URL creation failed", zap.Error(err))
		return "", err
	}

	return presignedURL.URL, nil
}

func (p *FilebaseProvider) Name() string {
	return string(domain.ProviderFilebase)
}

func LoadFilebaseConfigFromEnv() *FilebaseConfig {
	return &FilebaseConfig{
		AccessKey: os.Getenv("FILEBASE_ACCESS_KEY"),
		SecretKey: os.Getenv("FILEBASE_SECRET_KEY"),
		Endpoint:  "https://s3.filebase.com",
	}
}
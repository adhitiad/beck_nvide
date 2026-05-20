package repository

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type OCIProvider struct {
	client        objectstorage.ObjectStorageClient
	namespace     string
	compartmentID string
	region        string
	logger        *zap.Logger
}

type OCIConfig struct {
	Namespace     string
	CompartmentID string
	Region        string
	TenancyOCID   string
	UserOCID      string
	PrivateKey    string
	Fingerprint   string
}

func NewOCIProvider(cfg *OCIConfig, logger *zap.Logger) *OCIProvider {
	configProvider := common.NewRawConfigurationProvider(
		cfg.TenancyOCID,
		cfg.UserOCID,
		cfg.Region,
		cfg.Fingerprint,
		cfg.PrivateKey,
		nil,
	)

	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(configProvider)
	if err != nil {
		logger.Fatal("Failed to create OCI client", zap.Error(err))
	}

	return &OCIProvider{
		client:        client,
		namespace:     cfg.Namespace,
		compartmentID: cfg.CompartmentID,
		region:        cfg.Region,
		logger:        logger,
	}
}

func getReaderSize(body io.Reader) int64 {
	if seekable, ok := body.(io.Seeker); ok {
		if curr, err := seekable.Seek(0, io.SeekCurrent); err == nil {
			if end, err := seekable.Seek(0, io.SeekEnd); err == nil {
				_, _ = seekable.Seek(curr, io.SeekStart)
				return end - curr
			}
		}
	}
	return 0
}

func (p *OCIProvider) Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string) (*domain.UploadResult, error) {
	p.logger.Debug("OCI Upload", zap.String("bucket", bucket), zap.String("key", key))

	readCloser, ok := body.(io.ReadCloser)
	if !ok {
		readCloser = io.NopCloser(body)
	}

	size := getReaderSize(body)

	req := objectstorage.PutObjectRequest{
		NamespaceName: common.String(p.namespace),
		BucketName:    common.String(bucket),
		ObjectName:    common.String(key),
		PutObjectBody: readCloser,
		ContentType:   common.String(contentType),
		OpcMeta:       metadata,
	}

	resp, err := p.client.PutObject(ctx, req)
	if err != nil {
		p.logger.Error("OCI upload failed", zap.Error(err))
		return nil, err
	}

	url := "https://objectstorage." + p.region + ".oraclecloud.com/n/" + p.namespace + "/b/" + bucket + "/o/" + key

	etag := ""
	if resp.ETag != nil {
		etag = *resp.ETag
	}

	return &domain.UploadResult{
		URL:      url,
		Key:      key,
		Bucket:   bucket,
		Provider: string(domain.ProviderOCI),
		Size:     size,
		MD5:      etag,
		Metadata: metadata,
	}, nil
}

func (p *OCIProvider) UploadMultipart(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string, partSize int64) (*domain.UploadResult, error) {
	p.logger.Debug("OCI Multipart Upload", zap.String("bucket", bucket), zap.String("key", key))

	readCloser, ok := body.(io.ReadCloser)
	if !ok {
		readCloser = io.NopCloser(body)
	}

	size := getReaderSize(body)

	req := objectstorage.PutObjectRequest{
		NamespaceName: common.String(p.namespace),
		BucketName:    common.String(bucket),
		ObjectName:    common.String(key),
		PutObjectBody: readCloser,
		ContentType:   common.String(contentType),
		OpcMeta:       metadata,
	}

	resp, err := p.client.PutObject(ctx, req)
	if err != nil {
		p.logger.Error("OCI multipart upload failed", zap.Error(err))
		return nil, err
	}

	url := "https://objectstorage." + p.region + ".oraclecloud.com/n/" + p.namespace + "/b/" + bucket + "/o/" + key

	etag := ""
	if resp.ETag != nil {
		etag = *resp.ETag
	}

	return &domain.UploadResult{
		URL:      url,
		Key:      key,
		Bucket:   bucket,
		Provider: string(domain.ProviderOCI),
		Size:     size,
		MD5:      etag,
		Metadata: metadata,
	}, nil
}

func (p *OCIProvider) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	p.logger.Debug("OCI Download", zap.String("bucket", bucket), zap.String("key", key))

	req := objectstorage.GetObjectRequest{
		NamespaceName: common.String(p.namespace),
		BucketName:    common.String(bucket),
		ObjectName:    common.String(key),
	}

	resp, err := p.client.GetObject(ctx, req)
	if err != nil {
		p.logger.Error("OCI download failed", zap.Error(err))
		return nil, err
	}

	return resp.Content, nil
}

func (p *OCIProvider) Delete(ctx context.Context, bucket, key string) error {
	p.logger.Debug("OCI Delete", zap.String("bucket", bucket), zap.String("key", key))

	req := objectstorage.DeleteObjectRequest{
		NamespaceName: common.String(p.namespace),
		BucketName:    common.String(bucket),
		ObjectName:    common.String(key),
	}

	_, err := p.client.DeleteObject(ctx, req)
	if err != nil {
		p.logger.Error("OCI delete failed", zap.Error(err))
		return err
	}

	return nil
}

func (p *OCIProvider) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	p.logger.Debug("OCI GetPresignedURL", zap.String("bucket", bucket), zap.String("key", key))

	req := objectstorage.CreatePreauthenticatedRequestRequest{
		NamespaceName: common.String(p.namespace),
		BucketName:    common.String(bucket),
		CreatePreauthenticatedRequestDetails: objectstorage.CreatePreauthenticatedRequestDetails{
			Name:       common.String(key + "-presigned"),
			ObjectName: common.String(key),
			AccessType: objectstorage.CreatePreauthenticatedRequestDetailsAccessTypeObjectread,
			TimeExpires: &common.SDKTime{
				Time: time.Now().Add(expiry),
			},
		},
	}

	resp, err := p.client.CreatePreauthenticatedRequest(ctx, req)
	if err != nil {
		p.logger.Error("OCI presigned URL creation failed", zap.Error(err))
		return "", err
	}

	url := "https://objectstorage." + p.region + ".oraclecloud.com" + *resp.AccessUri
	return url, nil
}

func (p *OCIProvider) Name() string {
	return string(domain.ProviderOCI)
}

func LoadOCIConfigFromEnv() *OCIConfig {
	return &OCIConfig{
		Namespace:     os.Getenv("OCI_NAMESPACE"),
		CompartmentID: os.Getenv("OCI_COMPARTMENT_ID"),
		Region:        os.Getenv("OCI_REGION"),
		TenancyOCID:   os.Getenv("OCI_TENANCY_OCID"),
		UserOCID:      os.Getenv("OCI_USER_OCID"),
		PrivateKey:    os.Getenv("OCI_PRIVATE_KEY"),
		Fingerprint:   os.Getenv("OCI_FINGERPRINT"),
	}
}
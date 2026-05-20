package domain

import (
	"context"
	"io"
	"time"
)

type ContentType string

const (
	ContentTypeVideo         ContentType = "video"          // Stream Recording
	ContentTypeVOD           ContentType = "vod"            // VOD Standard
	ContentTypeVODPremium    ContentType = "vod_premium"    // VOD Premium
	ContentTypeThumbnail     ContentType = "image"          // Thumbnail/Gambar
	ContentTypePrivateCall   ContentType = "private_call"   // Private Call Recording
	ContentTypeBackup        ContentType = "backup"
	ContentTypeClip          ContentType = "clip"           // AI Clip
	ContentTypeKYC           ContentType = "kyc"
)

type StorageProviderType string

const (
	ProviderOCI      StorageProviderType = "oci"
	ProviderStorj    StorageProviderType = "storj"
	ProviderFilebase StorageProviderType = "filebase"
)

type StorageProvider interface {
	Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, metadata map[string]string) (*UploadResult, error)
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, bucket, key string) error
	GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	Name() string
}

type UploadResult struct {
	URL       string
	Key       string
	Bucket    string
	Provider  string
	Size      int64
	MD5       string
	IPFSHash  string
	Metadata  map[string]string
}

type StorageFile struct {
	ID           UUID
	Bucket       string
	Key          string
	Provider     string
	ReplicatedTo []string
	Size         int64
	ContentType  string
	Metadata     map[string]string
	CreatedAt    time.Time
}

type StorageReplicationJob struct {
	ID              UUID
	SourceKey       string
	SourceBucket    string
	SourceProvider  string
	TargetProviders []string
	RetryCount      int
	MaxRetries      int
	Status          ReplicationStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ReplicationStatus string

const (
	ReplicationStatusPending    ReplicationStatus = "pending"
	ReplicationStatusInProgress ReplicationStatus = "in_progress"
	ReplicationStatusCompleted  ReplicationStatus = "completed"
	ReplicationStatusFailed     ReplicationStatus = "failed"
)

type StorageReplicationRepository interface {
	Create(ctx context.Context, job *StorageReplicationJob) error
	GetByID(ctx context.Context, id UUID) (*StorageReplicationJob, error)
	Update(ctx context.Context, job *StorageReplicationJob) error
	ListPending(ctx context.Context, limit, offset int) ([]*StorageReplicationJob, error)
}

type StorageFileRepository interface {
	Create(ctx context.Context, file *StorageFile) error
	GetByID(ctx context.Context, id UUID) (*StorageFile, error)
	GetByKey(ctx context.Context, bucket, key string) (*StorageFile, error)
	UpdateReplicatedTo(ctx context.Context, id UUID, providers []string) error
	Delete(ctx context.Context, id UUID) error
	ListUnreplicated(ctx context.Context, limit int) ([]*StorageFile, error)
}
package domain

import (
	"context"
	"time"
)

// VOD visibility constants
const (
	VODVisibilityPublic    = "public"
	VODVisibilityFollowers = "followers"
	VODVisibilityPrivate   = "private"
)

// VOD status constants
const (
	VODStatusProcessing = "processing"
	VODStatusReady      = "ready"
	VODStatusFailed     = "failed"
)

// VODMedia represents a Video On Demand entity
type VODMedia struct {
	ID           UUID       `json:"id"`
	UserID       UUID       `json:"user_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	OriginalURL  string     `json:"original_url"`
	HLSURL       string     `json:"hls_url"`
	ThumbnailURL string     `json:"thumbnail_url"`
	Duration     int        `json:"duration"` // in seconds
	FileSize     int64      `json:"file_size"`
	Status       string     `json:"status"`
	Visibility   string     `json:"visibility"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"-"`

	// Relations
	User *User `json:"user,omitempty"`
}

// VODMediaRepository defines operations for VOD
type VODMediaRepository interface {
	Create(ctx context.Context, vod *VODMedia) error
	Update(ctx context.Context, vod *VODMedia) error
	GetByID(ctx context.Context, id UUID) (*VODMedia, error)
	ListByUser(ctx context.Context, userID UUID, limit, offset int) ([]*VODMedia, error)
	Delete(ctx context.Context, id UUID) error
}

// VODUseCase defines operations for VOD use case
type VODUseCaseInterface interface {
	// UploadVideo starts the async processing of an uploaded video
	UploadVideo(ctx context.Context, userID UUID, title, description, visibility, tempFilePath string, originalFileName string) (*VODMedia, error)
	GetVODDetail(ctx context.Context, id UUID) (*VODMedia, error)
	ListUserVODs(ctx context.Context, userID UUID, limit, offset int) ([]*VODMedia, error)
	UpdateVisibility(ctx context.Context, id UUID, userID UUID, visibility string) error
	DeleteVOD(ctx context.Context, id UUID, userID UUID) error
}

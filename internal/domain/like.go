package domain

import (
	"context"
	"time"
)

// Like represents a like on content (stream, vod, story, comment)
type Like struct {
	ID          UUID      `json:"id" db:"id"`
	UserID      UUID      `json:"user_id" db:"user_id"`
	ContentID   UUID      `json:"content_id" db:"content_id"`
	ContentType string    `json:"content_type" db:"content_type"` // "stream", "vod", "story", "comment"
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// LikeRepository defines the contract for like data access
type LikeRepository interface {
	Create(ctx context.Context, like *Like) error
	Delete(ctx context.Context, userID, contentID UUID, contentType string) error
	HasLiked(ctx context.Context, userID, contentID UUID, contentType string) (bool, error)
	CountByContent(ctx context.Context, contentID UUID, contentType string) (int, error)
	GetByUser(ctx context.Context, userID UUID, limit, offset int) ([]*Like, error)
	BatchCreate(ctx context.Context, likes []*Like) error
}

// LikeTarget represents a target for like operations
type LikeTarget struct {
	ContentID   UUID
	ContentType string
}

package domain

import (
	"context"
	"time"
)

// Comment represents a comment on content (stream, vod, story)
type Comment struct {
	ID          UUID      `json:"id" db:"id"`
	UserID      UUID      `json:"user_id" db:"user_id"`
	ContentID   UUID      `json:"content_id" db:"content_id"`         // ID of stream/vod/story
	ContentType string    `json:"content_type" db:"content_type"`     // "stream", "vod", "story"
	ParentID    *UUID     `json:"parent_id,omitempty" db:"parent_id"` // For nested comments (1 level)
	Content     string    `json:"content" db:"content"`
	LikeCount   int       `json:"like_count" db:"like_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Relations
	User    *User      `json:"user,omitempty"`
	Replies []*Comment `json:"replies,omitempty"`
}

// CommentRepository defines the contract for comment data access
type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	GetByID(ctx context.Context, id UUID) (*Comment, error)
	GetByContentID(ctx context.Context, contentID UUID, contentType string, limit, offset int) ([]*Comment, error)
	GetReplies(ctx context.Context, parentID UUID, limit, offset int) ([]*Comment, error)
	GetByUser(ctx context.Context, userID UUID, limit, offset int) ([]*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id UUID) error
	IncrementLikeCount(ctx context.Context, id UUID) error
	CountByContent(ctx context.Context, contentID UUID, contentType string) (int, error)
}

// CommentLike represents a like on a comment
type CommentLike struct {
	ID        UUID      `json:"id" db:"id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	CommentID UUID      `json:"comment_id" db:"comment_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CommentLikeRepository defines the contract for comment like data access
type CommentLikeRepository interface {
	Create(ctx context.Context, like *CommentLike) error
	Delete(ctx context.Context, userID, commentID UUID) error
	HasLiked(ctx context.Context, userID, commentID UUID) (bool, error)
	CountByCommentID(ctx context.Context, commentID UUID) (int, error)
}

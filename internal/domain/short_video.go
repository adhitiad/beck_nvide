package domain

import (
	"context"
	"time"
)

// Short video statuses
const (
	ShortVideoStatusActive     = "active"
	ShortVideoStatusHidden     = "hidden"
	ShortVideoStatusDeleted    = "deleted"
	ShortVideoStatusProcessing = "processing"
)

// ShortVideo represents a TikTok-like short video
type ShortVideo struct {
	ID           UUID      `json:"id"`
	UserID       UUID      `json:"user_id"`
	VideoURL     string    `json:"video_url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Caption      string    `json:"caption"`
	Duration     int       `json:"duration"` // seconds, max 60
	LikeCount    int       `json:"like_count"`
	CommentCount int       `json:"comment_count"`
	ShareCount   int       `json:"share_count"`
	ViewCount    int       `json:"view_count"`
	GiftValue    int64     `json:"gift_value"`
	Status       string    `json:"status"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	User    *User `json:"user,omitempty"`
	IsLiked bool  `json:"is_liked"` // virtual field for current viewer
}

// ShortVideoLike represents a like on a short video
type ShortVideoLike struct {
	ID        UUID      `json:"id"`
	VideoID   UUID      `json:"video_id"`
	UserID    UUID      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ShortVideoComment represents a comment on a short video
type ShortVideoComment struct {
	ID        UUID      `json:"id"`
	VideoID   UUID      `json:"video_id"`
	UserID    UUID      `json:"user_id"`
	Content   string    `json:"content"`
	ParentID  *UUID     `json:"parent_id,omitempty"`
	LikeCount int       `json:"like_count"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	User    *User                `json:"user,omitempty"`
	Replies []*ShortVideoComment `json:"replies,omitempty"`
}

// CreateShortVideoInput represents the input for creating a short video
type CreateShortVideoInput struct {
	VideoURL     string
	ThumbnailURL string
	Caption      string
	Duration     int
	Tags         []string
}

// ShortVideoRepository defines the contract for short video data access
type ShortVideoRepository interface {
	// Videos
	Create(ctx context.Context, video *ShortVideo) error
	GetByID(ctx context.Context, id UUID) (*ShortVideo, error)
	Update(ctx context.Context, video *ShortVideo) error
	Delete(ctx context.Context, id UUID) error
	GetFeed(ctx context.Context, viewerID UUID, limit, offset int) ([]*ShortVideo, error)
	GetByUserID(ctx context.Context, userID UUID, limit, offset int) ([]*ShortVideo, error)
	GetTrending(ctx context.Context, limit, offset int) ([]*ShortVideo, error)
	IncrementViewCount(ctx context.Context, id UUID) error
	IncrementShareCount(ctx context.Context, id UUID) error
	UpdateGiftValue(ctx context.Context, id UUID, amount int64) error

	// Likes
	Like(ctx context.Context, like *ShortVideoLike) error
	Unlike(ctx context.Context, videoID, userID UUID) error
	HasLiked(ctx context.Context, videoID, userID UUID) (bool, error)

	// Comments
	CreateComment(ctx context.Context, comment *ShortVideoComment) error
	GetComments(ctx context.Context, videoID UUID, limit, offset int) ([]*ShortVideoComment, error)
	GetCommentReplies(ctx context.Context, parentID UUID, limit, offset int) ([]*ShortVideoComment, error)
	DeleteComment(ctx context.Context, id UUID) error
}

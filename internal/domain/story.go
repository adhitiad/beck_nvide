package domain

import (
	"context"
	"time"
)

// Story represents a 24-hour story (like Instagram/Facebook stories)
type Story struct {
	ID        UUID      `json:"id" db:"id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`       // Text content or media URL
	MediaType string    `json:"media_type" db:"media_type"` // "text", "image", "video"
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	ViewCount int       `json:"view_count" db:"view_count"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	MediaURL  string    `json:"media_url" db:"media_url"`
	Caption   string    `json:"caption" db:"caption"`
	IsExpired bool      `json:"is_expired" db:"is_expired"`

	// Relations
	User *User `json:"user,omitempty"`
}

// HasExpired checks if story has expired (24h or flagged as expired)
func (s *Story) HasExpired() bool {
	return s.IsExpired || time.Now().After(s.ExpiresAt)
}

// StoryView represents a view of a story
type StoryView struct {
	ID       UUID      `json:"id" db:"id"`
	StoryID  UUID      `json:"story_id" db:"story_id"`
	UserID   UUID      `json:"user_id" db:"user_id"`
	ViewerID UUID      `json:"viewer_id" db:"viewer_id"`
	ViewedAt time.Time `json:"viewed_at" db:"viewed_at"`
}

// StoryRepository defines the contract for story data access
type StoryRepository interface {
	Create(ctx context.Context, story *Story) error
	GetByID(ctx context.Context, id UUID) (*Story, error)
	GetByUserID(ctx context.Context, userID UUID, limit, offset int) ([]*Story, error)
	GetActiveStories(ctx context.Context, userID UUID) ([]*Story, error)
	GetFeedStories(ctx context.Context, userIDs []UUID, limit int) ([]*Story, error)
	Update(ctx context.Context, story *Story) error
	Delete(ctx context.Context, id UUID) error
	DeleteExpired(ctx context.Context) error
	IncrementViewCount(ctx context.Context, id UUID) error
	AddViewCount(ctx context.Context, id UUID, delta int) error
	CountByUser(ctx context.Context, userID UUID) (int, error)
}

// StoryViewRepository defines the contract for story view data access
type StoryViewRepository interface {
	Create(ctx context.Context, view *StoryView) error
	GetByStoryID(ctx context.Context, storyID UUID) ([]*StoryView, error)
	GetByUserIDAndStoryID(ctx context.Context, userID, storyID UUID) (*StoryView, error)
	HasViewed(ctx context.Context, userID, storyID UUID) (bool, error)
	CountByStoryID(ctx context.Context, storyID UUID) (int, error)
}

package domain

import (
	"context"
	"time"
)

// StreamTag represents a tag that can be assigned to streams
type StreamTag struct {
	ID        UUID      `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"` // general, mood, content, special
	Keywords  []string  `json:"keywords"` // keywords that trigger this tag
	CreatedAt time.Time `json:"created_at"`
}

// StreamTagRepository defines the contract for stream tag data access
type StreamTagRepository interface {
	Create(ctx context.Context, tag *StreamTag) error
	GetByID(ctx context.Context, id UUID) (*StreamTag, error)
	GetByName(ctx context.Context, name string) (*StreamTag, error)
	ListAll(ctx context.Context) ([]*StreamTag, error)
	ListByCategory(ctx context.Context, category string) ([]*StreamTag, error)

	// Stream-Tag mapping
	AddTagToStream(ctx context.Context, streamID, tagID UUID) error
	RemoveTagFromStream(ctx context.Context, streamID, tagID UUID) error
	GetStreamTags(ctx context.Context, streamID UUID) ([]*StreamTag, error)
	ClearStreamTags(ctx context.Context, streamID UUID) error
}

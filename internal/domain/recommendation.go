package domain

import (
	"context"
	"time"
)

// UserInteraction merepresentasikan tindakan user terhadap stream/konten
type UserInteraction struct {
	ID              UUID                   `json:"id"`
	UserID          UUID                   `json:"user_id"`
	StreamID        *UUID                  `json:"stream_id,omitempty"`
	InteractionType string                 `json:"interaction_type"` // 'watch', 'like', 'comment', 'gift'
	DurationSeconds int                    `json:"duration_seconds"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
}

// RecommendationRepository mendefinisikan operasi DB untuk sistem rekomendasi AI
type RecommendationRepository interface {
	SaveInteraction(ctx context.Context, interaction *UserInteraction) error
	GetUserInteractions(ctx context.Context, userID UUID, limit int) ([]*UserInteraction, error)
	GetHostPreferenceVector(ctx context.Context, userID UUID) (map[UUID]float64, error)
	GetCategoryPreferenceVector(ctx context.Context, userID UUID) (map[string]float64, error)
}

// RecommendationUseCaseInterface mendefinisikan operasi bisnis untuk sistem rekomendasi AI
type RecommendationUseCaseInterface interface {
	TrackInteraction(ctx context.Context, userID UUID, streamID *UUID, interactionType string, duration int, metadata map[string]interface{}) error
	GetRecommendedStreams(ctx context.Context, userID UUID, limit int) ([]*Stream, error)
	GetRecommendedVODs(ctx context.Context, userID UUID, limit int) ([]*VODMedia, error)
}

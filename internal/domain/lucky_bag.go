package domain

import (
	"context"
	"time"
)

// Lucky bag statuses
const (
	LuckyBagStatusActive   = "active"
	LuckyBagStatusDepleted = "depleted"
	LuckyBagStatusExpired  = "expired"
)

// LuckyBag represents a random gift box created by a host during stream
type LuckyBag struct {
	ID        UUID       `json:"id"`
	HostID    UUID       `json:"host_id"`
	StreamID  UUID       `json:"stream_id"`
	MinValue  int64      `json:"min_value"`
	MaxValue  int64      `json:"max_value"`
	TotalCount int       `json:"total_count"`
	Remaining  int       `json:"remaining"`
	TotalPool  int64     `json:"total_pool"`
	Status     string    `json:"status"` // active, depleted, expired
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// LuckyBagClaim represents a user's claim on a lucky bag
type LuckyBagClaim struct {
	ID        UUID      `json:"id"`
	BagID     UUID      `json:"bag_id"`
	UserID    UUID      `json:"user_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

// LuckyBagRepository defines the contract for lucky bag data access
type LuckyBagRepository interface {
	Create(ctx context.Context, bag *LuckyBag) error
	GetByID(ctx context.Context, id UUID) (*LuckyBag, error)
	GetActiveByStream(ctx context.Context, streamID UUID) ([]*LuckyBag, error)
	DecrementRemaining(ctx context.Context, id UUID) error
	UpdateStatus(ctx context.Context, id UUID, status string) error

	// Claims
	CreateClaim(ctx context.Context, claim *LuckyBagClaim) error
	HasClaimed(ctx context.Context, bagID, userID UUID) (bool, error)
	GetClaimsByBag(ctx context.Context, bagID UUID) ([]*LuckyBagClaim, error)
}

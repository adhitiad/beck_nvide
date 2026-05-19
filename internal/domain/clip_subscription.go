package domain

import (
	"context"
	"time"
)

// ClipSubscriptionPlan represents a subscription tier
type ClipSubscriptionPlan struct {
	ID           UUID      `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"` // 'VIP1', 'VIP2', 'VIP3'
	Price        int64     `json:"price" db:"price"`
	Quota        int       `json:"quota" db:"quota"`
	DurationDays int       `json:"duration_days" db:"duration_days"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ClipSubscription represents an active subscription for a user
type ClipSubscription struct {
	ID         UUID      `json:"id" db:"id"`
	UserID     UUID      `json:"user_id" db:"user_id"`
	PlanID     UUID      `json:"plan_id" db:"plan_id"`
	StartDate  time.Time `json:"start_date" db:"start_date"`
	EndDate    time.Time `json:"end_date" db:"end_date"`
	QuotaUsed  int       `json:"quota_used" db:"quota_used"`
	QuotaTotal int       `json:"quota_total" db:"quota_total"`
	Status     string    `json:"status" db:"status"` // 'active', 'expired'
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ClipGenerationLog tracks who and when generated a clip under what subscription
type ClipGenerationLog struct {
	ID             UUID      `json:"id" db:"id"`
	UserID         UUID      `json:"user_id" db:"user_id"`
	StreamID       UUID      `json:"stream_id" db:"stream_id"`
	SubscriptionID *UUID     `json:"subscription_id,omitempty" db:"subscription_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ClipSubscriptionRepository defines data access methods for subscriptions
type ClipSubscriptionRepository interface {
	ListPlans(ctx context.Context) ([]*ClipSubscriptionPlan, error)
	GetPlanByID(ctx context.Context, planID UUID) (*ClipSubscriptionPlan, error)
	GetPlanByName(ctx context.Context, name string) (*ClipSubscriptionPlan, error)
	
	CreateSubscription(ctx context.Context, sub *ClipSubscription) error
	GetActiveSubscription(ctx context.Context, userID UUID) (*ClipSubscription, error)
	GetSubscriptionByID(ctx context.Context, id UUID) (*ClipSubscription, error)
	UpdateSubscriptionQuota(ctx context.Context, id UUID, quotaUsed int) error
	UpdateSubscriptionStatus(ctx context.Context, id UUID, status string) error
	ListSubscriptionHistory(ctx context.Context, userID UUID, limit, offset int) ([]*ClipSubscription, error)
	HasSubscribedBefore(ctx context.Context, userID UUID) (bool, error)

	CreateGenerationLog(ctx context.Context, log *ClipGenerationLog) error
	CountGeneratedClipsThisMonth(ctx context.Context, userID UUID) (int, error)
}

package domain

import (
	"context"
	"time"
)

type PushSubscription struct {
	ID        UUID      `json:"id" db:"id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Endpoint  string    `json:"endpoint" db:"endpoint"`
	P256DHKey string    `json:"p256dh_key" db:"p256dh_key"`
	AuthKey   string    `json:"auth_key" db:"auth_key"`
	Topics    []string  `json:"topics" db:"topics"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type PushSubscriptionRepository interface {
	Upsert(ctx context.Context, item *PushSubscription) error
	DeleteByEndpointAndUserID(ctx context.Context, userID UUID, endpoint string) error
	ListByUserID(ctx context.Context, userID UUID) ([]*PushSubscription, error)
}

type PushNotificationUsecase interface {
	Subscribe(ctx context.Context, userID UUID, endpoint, p256dh, auth string, topics []string) error
	Unsubscribe(ctx context.Context, userID UUID, endpoint string) error
	SendTest(ctx context.Context, userID UUID, title, body string) (int, error)
}

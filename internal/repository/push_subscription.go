package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type pushSubscriptionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewPushSubscriptionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PushSubscriptionRepository {
	return &pushSubscriptionRepository{db: db, logger: logger}
}

func (r *pushSubscriptionRepository) Upsert(ctx context.Context, item *domain.PushSubscription) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh_key, auth_key, topics, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,NOW(),NOW())
		ON CONFLICT (user_id, endpoint)
		DO UPDATE SET
			p256dh_key = EXCLUDED.p256dh_key,
			auth_key = EXCLUDED.auth_key,
			topics = EXCLUDED.topics,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`, item.ID, item.UserID, item.Endpoint, item.P256DHKey, item.AuthKey, item.Topics).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
}

func (r *pushSubscriptionRepository) DeleteByEndpointAndUserID(ctx context.Context, userID domain.UUID, endpoint string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	return err
}

func (r *pushSubscriptionRepository) ListByUserID(ctx context.Context, userID domain.UUID) ([]*domain.PushSubscription, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, endpoint, p256dh_key, auth_key, topics, created_at, updated_at
		FROM push_subscriptions
		WHERE user_id = $1
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domain.PushSubscription
	for rows.Next() {
		var s domain.PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256DHKey, &s.AuthKey, &s.Topics, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, &s)
	}
	return items, nil
}

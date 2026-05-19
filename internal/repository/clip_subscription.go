package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type clipSubscriptionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewClipSubscriptionRepository creates new clip subscription repository
func NewClipSubscriptionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.ClipSubscriptionRepository {
	return &clipSubscriptionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *clipSubscriptionRepository) ListPlans(ctx context.Context) ([]*domain.ClipSubscriptionPlan, error) {
	query := `
		SELECT id, name, price, quota, duration_days, created_at
		FROM clip_subscription_plans
		ORDER BY price ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.logger.Error("Failed to list plans", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	plans := make([]*domain.ClipSubscriptionPlan, 0)
	for rows.Next() {
		plan := &domain.ClipSubscriptionPlan{}
		err := rows.Scan(&plan.ID, &plan.Name, &plan.Price, &plan.Quota, &plan.DurationDays, &plan.CreatedAt)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

func (r *clipSubscriptionRepository) GetPlanByID(ctx context.Context, planID domain.UUID) (*domain.ClipSubscriptionPlan, error) {
	query := `
		SELECT id, name, price, quota, duration_days, created_at
		FROM clip_subscription_plans
		WHERE id = $1
	`
	plan := &domain.ClipSubscriptionPlan{}
	err := r.db.QueryRow(ctx, query, planID).Scan(&plan.ID, &plan.Name, &plan.Price, &plan.Quota, &plan.DurationDays, &plan.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get plan by ID", zap.Error(err), zap.String("id", planID.String()))
		return nil, err
	}

	return plan, nil
}

func (r *clipSubscriptionRepository) GetPlanByName(ctx context.Context, name string) (*domain.ClipSubscriptionPlan, error) {
	query := `
		SELECT id, name, price, quota, duration_days, created_at
		FROM clip_subscription_plans
		WHERE name = $1
	`
	plan := &domain.ClipSubscriptionPlan{}
	err := r.db.QueryRow(ctx, query, name).Scan(&plan.ID, &plan.Name, &plan.Price, &plan.Quota, &plan.DurationDays, &plan.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get plan by name", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return plan, nil
}

func (r *clipSubscriptionRepository) CreateSubscription(ctx context.Context, sub *domain.ClipSubscription) error {
	query := `
		INSERT INTO clip_subscriptions (id, user_id, plan_id, start_date, end_date, quota_used, quota_total, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING created_at
	`
	var createdAt time.Time
	err := r.db.QueryRow(ctx, query,
		sub.ID,
		sub.UserID,
		sub.PlanID,
		sub.StartDate,
		sub.EndDate,
		sub.QuotaUsed,
		sub.QuotaTotal,
		sub.Status,
	).Scan(&createdAt)

	if err != nil {
		r.logger.Error("Failed to create subscription", zap.Error(err), zap.String("user_id", sub.UserID.String()))
		return err
	}

	sub.CreatedAt = createdAt
	return nil
}

func (r *clipSubscriptionRepository) GetActiveSubscription(ctx context.Context, userID domain.UUID) (*domain.ClipSubscription, error) {
	// First, check and update expired subscriptions to 'expired' status
	updateQuery := `
		UPDATE clip_subscriptions
		SET status = 'expired'
		WHERE user_id = $1 AND status = 'active' AND end_date < NOW()
	`
	_, _ = r.db.Exec(ctx, updateQuery, userID)

	// Fetch active subscription
	query := `
		SELECT id, user_id, plan_id, start_date, end_date, quota_used, quota_total, status, created_at
		FROM clip_subscriptions
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at DESC
		LIMIT 1
	`
	sub := &domain.ClipSubscription{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.StartDate, &sub.EndDate, &sub.QuotaUsed, &sub.QuotaTotal, &sub.Status, &sub.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get active subscription", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}

	return sub, nil
}

func (r *clipSubscriptionRepository) GetSubscriptionByID(ctx context.Context, id domain.UUID) (*domain.ClipSubscription, error) {
	query := `
		SELECT id, user_id, plan_id, start_date, end_date, quota_used, quota_total, status, created_at
		FROM clip_subscriptions
		WHERE id = $1
	`
	sub := &domain.ClipSubscription{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.StartDate, &sub.EndDate, &sub.QuotaUsed, &sub.QuotaTotal, &sub.Status, &sub.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get subscription by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	return sub, nil
}

func (r *clipSubscriptionRepository) UpdateSubscriptionQuota(ctx context.Context, id domain.UUID, quotaUsed int) error {
	query := `
		UPDATE clip_subscriptions
		SET quota_used = $1
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, quotaUsed, id)
	if err != nil {
		r.logger.Error("Failed to update subscription quota", zap.Error(err), zap.String("id", id.String()))
		return err
	}
	return nil
}

func (r *clipSubscriptionRepository) UpdateSubscriptionStatus(ctx context.Context, id domain.UUID, status string) error {
	query := `
		UPDATE clip_subscriptions
		SET status = $1
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		r.logger.Error("Failed to update subscription status", zap.Error(err), zap.String("id", id.String()))
		return err
	}
	return nil
}

func (r *clipSubscriptionRepository) ListSubscriptionHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.ClipSubscription, error) {
	query := `
		SELECT id, user_id, plan_id, start_date, end_date, quota_used, quota_total, status, created_at
		FROM clip_subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list subscription history", zap.Error(err), zap.String("user_id", userID.String()))
		return nil, err
	}
	defer rows.Close()

	history := make([]*domain.ClipSubscription, 0)
	for rows.Next() {
		sub := &domain.ClipSubscription{}
		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.StartDate, &sub.EndDate, &sub.QuotaUsed, &sub.QuotaTotal, &sub.Status, &sub.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, sub)
	}

	return history, nil
}

func (r *clipSubscriptionRepository) HasSubscribedBefore(ctx context.Context, userID domain.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM clip_subscriptions WHERE user_id = $1
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		r.logger.Error("Failed to check if user has subscribed before", zap.Error(err), zap.String("user_id", userID.String()))
		return false, err
	}
	return exists, nil
}

func (r *clipSubscriptionRepository) CreateGenerationLog(ctx context.Context, log *domain.ClipGenerationLog) error {
	query := `
		INSERT INTO clip_generation_logs (id, user_id, stream_id, subscription_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING created_at
	`
	var createdAt time.Time
	err := r.db.QueryRow(ctx, query,
		log.ID,
		log.UserID,
		log.StreamID,
		log.SubscriptionID,
	).Scan(&createdAt)

	if err != nil {
		r.logger.Error("Failed to create clip generation log", zap.Error(err), zap.String("user_id", log.UserID.String()))
		return err
	}

	log.CreatedAt = createdAt
	return nil
}

func (r *clipSubscriptionRepository) CountGeneratedClipsThisMonth(ctx context.Context, userID domain.UUID) (int, error) {
	query := `
		SELECT COUNT(1)
		FROM clip_generation_logs
		WHERE user_id = $1 AND created_at >= NOW() - INTERVAL '30 days'
	`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to count generated clips", zap.Error(err), zap.String("user_id", userID.String()))
		return 0, err
	}
	return count, nil
}

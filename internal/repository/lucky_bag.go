package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type luckyBagRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewLuckyBagRepository(db *pgxpool.Pool, logger *zap.Logger) domain.LuckyBagRepository {
	return &luckyBagRepository{db: db, logger: logger}
}

func (r *luckyBagRepository) Create(ctx context.Context, bag *domain.LuckyBag) error {
	query := `INSERT INTO lucky_bags (id, host_id, stream_id, min_value, max_value, total_count,
		remaining, total_pool, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10) RETURNING created_at`
	return r.db.QueryRow(ctx, query, bag.ID, bag.HostID, bag.StreamID, bag.MinValue,
		bag.MaxValue, bag.TotalCount, bag.Remaining, bag.TotalPool, bag.Status, bag.ExpiresAt).
		Scan(&bag.CreatedAt)
}

func (r *luckyBagRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.LuckyBag, error) {
	query := `SELECT id, host_id, stream_id, min_value, max_value, total_count, remaining,
		total_pool, status, created_at, expires_at
		FROM lucky_bags WHERE id=$1`
	var bag domain.LuckyBag
	err := r.db.QueryRow(ctx, query, id).Scan(&bag.ID, &bag.HostID, &bag.StreamID,
		&bag.MinValue, &bag.MaxValue, &bag.TotalCount, &bag.Remaining, &bag.TotalPool,
		&bag.Status, &bag.CreatedAt, &bag.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &bag, nil
}

func (r *luckyBagRepository) GetActiveByStream(ctx context.Context, streamID domain.UUID) ([]*domain.LuckyBag, error) {
	query := `SELECT id, host_id, stream_id, min_value, max_value, total_count, remaining,
		total_pool, status, created_at, expires_at
		FROM lucky_bags WHERE stream_id=$1 AND status='active' AND remaining > 0`
	rows, err := r.db.Query(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LuckyBag
	for rows.Next() {
		var bag domain.LuckyBag
		if err := rows.Scan(&bag.ID, &bag.HostID, &bag.StreamID, &bag.MinValue, &bag.MaxValue,
			&bag.TotalCount, &bag.Remaining, &bag.TotalPool, &bag.Status, &bag.CreatedAt, &bag.ExpiresAt); err != nil {
			return nil, err
		}
		list = append(list, &bag)
	}
	return list, nil
}

func (r *luckyBagRepository) DecrementRemaining(ctx context.Context, id domain.UUID) error {
	result, err := r.db.Exec(ctx,
		`UPDATE lucky_bags SET remaining = remaining - 1,
		status = CASE WHEN remaining - 1 <= 0 THEN 'depleted' ELSE status END
		WHERE id=$1 AND remaining > 0`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrCodeConflict, "lucky bag is depleted", nil)
	}
	return nil
}

func (r *luckyBagRepository) UpdateStatus(ctx context.Context, id domain.UUID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE lucky_bags SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *luckyBagRepository) CreateClaim(ctx context.Context, claim *domain.LuckyBagClaim) error {
	query := `INSERT INTO lucky_bag_claims (id, bag_id, user_id, amount, created_at)
		VALUES ($1, $2, $3, $4, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, claim.ID, claim.BagID, claim.UserID, claim.Amount).
		Scan(&claim.CreatedAt)
}

func (r *luckyBagRepository) HasClaimed(ctx context.Context, bagID, userID domain.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM lucky_bag_claims WHERE bag_id=$1 AND user_id=$2)`,
		bagID, userID).Scan(&exists)
	return exists, err
}

func (r *luckyBagRepository) GetClaimsByBag(ctx context.Context, bagID domain.UUID) ([]*domain.LuckyBagClaim, error) {
	query := `SELECT id, bag_id, user_id, amount, created_at FROM lucky_bag_claims WHERE bag_id=$1 ORDER BY created_at`
	rows, err := r.db.Query(ctx, query, bagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.LuckyBagClaim
	for rows.Next() {
		var c domain.LuckyBagClaim
		if err := rows.Scan(&c.ID, &c.BagID, &c.UserID, &c.Amount, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &c)
	}
	return list, nil
}

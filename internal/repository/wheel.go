package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type wheelRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewWheelRepository(db *pgxpool.Pool, logger *zap.Logger) domain.WheelRepository {
	return &wheelRepository{db: db, logger: logger}
}

func (r *wheelRepository) CreatePrize(ctx context.Context, prize *domain.WheelPrize) error {
	query := `INSERT INTO wheel_prizes (id, name, type, value, item_id, icon_url, probability, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, prize.ID, prize.Name, prize.Type, prize.Value,
		prize.ItemID, prize.IconURL, prize.Probability, prize.IsActive).Scan(&prize.CreatedAt)
}

func (r *wheelRepository) ListActivePrizes(ctx context.Context) ([]*domain.WheelPrize, error) {
	query := `SELECT id, name, type, value, item_id, icon_url, probability, is_active, created_at
		FROM wheel_prizes WHERE is_active=true ORDER BY probability DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.WheelPrize
	for rows.Next() {
		var p domain.WheelPrize
		if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Value, &p.ItemID, &p.IconURL,
			&p.Probability, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &p)
	}
	return list, nil
}

func (r *wheelRepository) GetPrizeByID(ctx context.Context, id domain.UUID) (*domain.WheelPrize, error) {
	query := `SELECT id, name, type, value, item_id, icon_url, probability, is_active, created_at
		FROM wheel_prizes WHERE id=$1`
	var p domain.WheelPrize
	err := r.db.QueryRow(ctx, query, id).Scan(&p.ID, &p.Name, &p.Type, &p.Value, &p.ItemID,
		&p.IconURL, &p.Probability, &p.IsActive, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *wheelRepository) UpdatePrize(ctx context.Context, prize *domain.WheelPrize) error {
	query := `UPDATE wheel_prizes SET name=$1, type=$2, value=$3, item_id=$4, icon_url=$5,
		probability=$6, is_active=$7 WHERE id=$8`
	_, err := r.db.Exec(ctx, query, prize.Name, prize.Type, prize.Value, prize.ItemID,
		prize.IconURL, prize.Probability, prize.IsActive, prize.ID)
	return err
}

func (r *wheelRepository) RecordSpin(ctx context.Context, spin *domain.WheelSpin) error {
	query := `INSERT INTO wheel_spins (id, user_id, prize_id, cost, created_at)
		VALUES ($1, $2, $3, $4, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, spin.ID, spin.UserID, spin.PrizeID, spin.Cost).
		Scan(&spin.CreatedAt)
}

func (r *wheelRepository) GetUserSpinsToday(ctx context.Context, userID domain.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM wheel_spins WHERE user_id=$1 AND created_at::date = CURRENT_DATE`,
		userID).Scan(&count)
	return count, err
}

func (r *wheelRepository) GetUserSpinHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.WheelSpin, error) {
	query := `SELECT ws.id, ws.user_id, ws.prize_id, ws.cost, ws.created_at,
		wp.id, wp.name, wp.type, wp.value, wp.item_id, wp.icon_url, wp.probability, wp.is_active, wp.created_at
		FROM wheel_spins ws JOIN wheel_prizes wp ON ws.prize_id = wp.id
		WHERE ws.user_id=$1 ORDER BY ws.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.WheelSpin
	for rows.Next() {
		var s domain.WheelSpin
		var p domain.WheelPrize
		if err := rows.Scan(&s.ID, &s.UserID, &s.PrizeID, &s.Cost, &s.CreatedAt,
			&p.ID, &p.Name, &p.Type, &p.Value, &p.ItemID, &p.IconURL, &p.Probability, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, err
		}
		s.Prize = &p
		list = append(list, &s)
	}
	return list, nil
}

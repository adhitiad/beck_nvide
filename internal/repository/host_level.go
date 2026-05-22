package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type hostLevelRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewHostLevelRepository(db *pgxpool.Pool, logger *zap.Logger) domain.HostLevelRepository {
	return &hostLevelRepository{db: db, logger: logger}
}

func (r *hostLevelRepository) ListAll(ctx context.Context) ([]*domain.HostLevel, error) {
	query := `SELECT id, name, display_name, min_stream_hours, min_total_income, commission_rate,
		badge_url, perks, sort_order, created_at
		FROM host_levels ORDER BY sort_order ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.HostLevel
	for rows.Next() {
		var hl domain.HostLevel
		if err := rows.Scan(&hl.ID, &hl.Name, &hl.DisplayName, &hl.MinStreamHours,
			&hl.MinTotalIncome, &hl.CommissionRate, &hl.BadgeURL, &hl.Perks,
			&hl.SortOrder, &hl.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &hl)
	}
	return list, nil
}

func (r *hostLevelRepository) GetByName(ctx context.Context, name string) (*domain.HostLevel, error) {
	query := `SELECT id, name, display_name, min_stream_hours, min_total_income, commission_rate,
		badge_url, perks, sort_order, created_at FROM host_levels WHERE name=$1`
	var hl domain.HostLevel
	err := r.db.QueryRow(ctx, query, name).Scan(&hl.ID, &hl.Name, &hl.DisplayName,
		&hl.MinStreamHours, &hl.MinTotalIncome, &hl.CommissionRate, &hl.BadgeURL,
		&hl.Perks, &hl.SortOrder, &hl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &hl, nil
}

func (r *hostLevelRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.HostLevel, error) {
	query := `SELECT id, name, display_name, min_stream_hours, min_total_income, commission_rate,
		badge_url, perks, sort_order, created_at FROM host_levels WHERE id=$1`
	var hl domain.HostLevel
	err := r.db.QueryRow(ctx, query, id).Scan(&hl.ID, &hl.Name, &hl.DisplayName,
		&hl.MinStreamHours, &hl.MinTotalIncome, &hl.CommissionRate, &hl.BadgeURL,
		&hl.Perks, &hl.SortOrder, &hl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &hl, nil
}

func (r *hostLevelRepository) GetEligibleLevel(ctx context.Context, streamHours int, totalIncome int64) (*domain.HostLevel, error) {
	query := `SELECT id, name, display_name, min_stream_hours, min_total_income, commission_rate,
		badge_url, perks, sort_order, created_at
		FROM host_levels
		WHERE min_stream_hours <= $1 AND min_total_income <= $2
		ORDER BY sort_order DESC LIMIT 1`
	var hl domain.HostLevel
	err := r.db.QueryRow(ctx, query, streamHours, totalIncome).Scan(&hl.ID, &hl.Name,
		&hl.DisplayName, &hl.MinStreamHours, &hl.MinTotalIncome, &hl.CommissionRate,
		&hl.BadgeURL, &hl.Perks, &hl.SortOrder, &hl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &hl, nil
}

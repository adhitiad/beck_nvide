package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

// AgencyCommission tracks MLM commission records
type AgencyCommission struct {
	ID          domain.UUID `json:"id"`
	AgencyID    domain.UUID `json:"agency_id"`
	FromHostID  domain.UUID `json:"from_host_id"`
	ToHostID    domain.UUID `json:"to_host_id"`
	Level       int         `json:"level"`
	Amount      int64       `json:"amount"`
	Percentage  int         `json:"percentage"`
	SourceTxID  *domain.UUID `json:"source_tx_id,omitempty"`
}

type agencyCommissionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewAgencyCommissionRepository(db *pgxpool.Pool, logger *zap.Logger) *agencyCommissionRepository {
	return &agencyCommissionRepository{db: db, logger: logger}
}

func (r *agencyCommissionRepository) Create(ctx context.Context, c *AgencyCommission) error {
	query := `INSERT INTO agency_commissions (id, agency_id, from_host_id, to_host_id, level, amount, percentage, source_tx_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`
	_, err := r.db.Exec(ctx, query, c.ID, c.AgencyID, c.FromHostID, c.ToHostID,
		c.Level, c.Amount, c.Percentage, c.SourceTxID)
	return err
}

func (r *agencyCommissionRepository) GetReferrer(ctx context.Context, hostID domain.UUID) (*domain.UUID, error) {
	var referrerID *domain.UUID
	err := r.db.QueryRow(ctx,
		`SELECT referrer_host_id FROM agency_hosts WHERE host_id=$1 AND status='active'`,
		hostID).Scan(&referrerID)
	if err != nil {
		return nil, err
	}
	return referrerID, nil
}

func (r *agencyCommissionRepository) SetReferrer(ctx context.Context, hostID, referrerID domain.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE agency_hosts SET referrer_host_id=$1 WHERE host_id=$2`,
		referrerID, hostID)
	return err
}

func (r *agencyCommissionRepository) GetCommissionsByHost(ctx context.Context, hostID domain.UUID, limit, offset int) ([]*AgencyCommission, error) {
	query := `SELECT id, agency_id, from_host_id, to_host_id, level, amount, percentage, source_tx_id
		FROM agency_commissions WHERE to_host_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, hostID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*AgencyCommission
	for rows.Next() {
		var c AgencyCommission
		if err := rows.Scan(&c.ID, &c.AgencyID, &c.FromHostID, &c.ToHostID,
			&c.Level, &c.Amount, &c.Percentage, &c.SourceTxID); err != nil {
			return nil, err
		}
		list = append(list, &c)
	}
	return list, nil
}

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type withdrawalRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewWithdrawalRepository(db *pgxpool.Pool, logger *zap.Logger) domain.WithdrawalRepository {
	return &withdrawalRepository{
		db:     db,
		logger: logger,
	}
}

func (r *withdrawalRepository) Create(ctx context.Context, w *domain.Withdrawal) error {
	query := `
		INSERT INTO withdrawals (
			id, user_id, amount_requested, gross_amount, 
			fee_platform, fee_processing, fee_tax, fee_agency, 
			total_fee, net_amount, agency_id, status, 
			payment_method, bank_account_info, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query,
		w.ID, w.UserID, w.AmountRequested, w.GrossAmount,
		w.FeePlatform, w.FeeProcessing, w.FeeTax, w.FeeAgency,
		w.TotalFee, w.NetAmount, w.AgencyID, w.Status,
		w.PaymentMethod, w.BankAccountInfo,
	)
	return err
}

func (r *withdrawalRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Withdrawal, error) {
	query := `SELECT * FROM withdrawals WHERE id = $1`
	w := &domain.Withdrawal{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&w.ID, &w.UserID, &w.AmountRequested, &w.GrossAmount,
		&w.FeePlatform, &w.FeeProcessing, &w.FeeTax, &w.FeeAgency,
		&w.TotalFee, &w.NetAmount, &w.AgencyID, &w.Status,
		&w.PaymentMethod, &w.BankAccountInfo, &w.TxReference,
		&w.ApprovedBy, &w.ApprovedAt, &w.CompletedAt,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *withdrawalRepository) List(ctx context.Context, userID *domain.UUID, status string, limit, offset int) ([]*domain.Withdrawal, error) {
	query := `SELECT * FROM withdrawals WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if userID != nil {
		query += " AND user_id = $" + string(rune('0'+argIdx))
		args = append(args, *userID)
		argIdx++
	}
	if status != "" {
		query += " AND status = $" + string(rune('0'+argIdx))
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY created_at DESC LIMIT $" + string(rune('0'+argIdx)) + " OFFSET $" + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Withdrawal
	for rows.Next() {
		w := &domain.Withdrawal{}
		err := rows.Scan(
			&w.ID, &w.UserID, &w.AmountRequested, &w.GrossAmount,
			&w.FeePlatform, &w.FeeProcessing, &w.FeeTax, &w.FeeAgency,
			&w.TotalFee, &w.NetAmount, &w.AgencyID, &w.Status,
			&w.PaymentMethod, &w.BankAccountInfo, &w.TxReference,
			&w.ApprovedBy, &w.ApprovedAt, &w.CompletedAt,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, nil
}

func (r *withdrawalRepository) UpdateStatus(ctx context.Context, id domain.UUID, status string, adminID *domain.UUID) error {
	query := `UPDATE withdrawals SET status = $1, approved_by = $2, approved_at = NOW(), updated_at = NOW() WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, adminID, id)
	return err
}

func (r *withdrawalRepository) GetActiveFeeRules(ctx context.Context) ([]*domain.FeeRule, error) {
	query := `SELECT id, name, fee_type, value, applies_to, is_active, priority, created_at FROM fee_rules WHERE is_active = true ORDER BY priority ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.FeeRule
	for rows.Next() {
		rule := &domain.FeeRule{}
		err := rows.Scan(&rule.ID, &rule.Name, &rule.FeeType, &rule.Value, &rule.AppliesTo, &rule.IsActive, &rule.Priority, &rule.CreatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *withdrawalRepository) CreateFeeAudit(ctx context.Context, audit *domain.WithdrawalFeeAudit) error {
	query := `INSERT INTO withdrawal_fee_audits (id, withdrawal_id, fee_name, fee_percentage, fee_amount, calculated_from, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW())`
	_, err := r.db.Exec(ctx, query, audit.ID, audit.WithdrawalID, audit.FeeName, audit.FeePercentage, audit.FeeAmount, audit.CalculatedFrom)
	return err
}

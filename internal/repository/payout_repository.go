package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type payoutRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewPayoutMethodRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PayoutMethodRepository {
	return &payoutRepository{db: db, logger: logger}
}

func (r *payoutRepository) ListByUserID(ctx context.Context, userID domain.UUID) ([]*domain.PayoutMethod, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, type, is_primary, bank_name, account_number, account_holder_name,
		       ewallet_provider, ewallet_phone_number, is_verified, micro_deposit_required, created_at, updated_at
		FROM payout_methods
		WHERE user_id = $1
		ORDER BY is_primary DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.PayoutMethod
	for rows.Next() {
		var m domain.PayoutMethod
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Type, &m.IsPrimary, &m.BankName, &m.AccountNumber, &m.AccountHolderName,
			&m.EwalletProvider, &m.EwalletPhoneNumber, &m.IsVerified, &m.MicroDepositRequired, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, nil
}

func (r *payoutRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.PayoutMethod, error) {
	var m domain.PayoutMethod
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, is_primary, bank_name, account_number, account_holder_name,
		       ewallet_provider, ewallet_phone_number, is_verified, micro_deposit_required, created_at, updated_at
		FROM payout_methods WHERE id = $1
	`, id).Scan(
		&m.ID, &m.UserID, &m.Type, &m.IsPrimary, &m.BankName, &m.AccountNumber, &m.AccountHolderName,
		&m.EwalletProvider, &m.EwalletPhoneNumber, &m.IsVerified, &m.MicroDepositRequired, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *payoutRepository) GetByIDAndUserID(ctx context.Context, id, userID domain.UUID) (*domain.PayoutMethod, error) {
	var m domain.PayoutMethod
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, is_primary, bank_name, account_number, account_holder_name,
		       ewallet_provider, ewallet_phone_number, is_verified, micro_deposit_required, created_at, updated_at
		FROM payout_methods WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(
		&m.ID, &m.UserID, &m.Type, &m.IsPrimary, &m.BankName, &m.AccountNumber, &m.AccountHolderName,
		&m.EwalletProvider, &m.EwalletPhoneNumber, &m.IsVerified, &m.MicroDepositRequired, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *payoutRepository) GetPrimaryByUserID(ctx context.Context, userID domain.UUID) (*domain.PayoutMethod, error) {
	var m domain.PayoutMethod
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, is_primary, bank_name, account_number, account_holder_name,
		       ewallet_provider, ewallet_phone_number, is_verified, micro_deposit_required, created_at, updated_at
		FROM payout_methods WHERE user_id = $1 AND is_primary = true
		LIMIT 1
	`, userID).Scan(
		&m.ID, &m.UserID, &m.Type, &m.IsPrimary, &m.BankName, &m.AccountNumber, &m.AccountHolderName,
		&m.EwalletProvider, &m.EwalletPhoneNumber, &m.IsVerified, &m.MicroDepositRequired, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *payoutRepository) CountByUserIDAndType(ctx context.Context, userID domain.UUID, payoutType string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(1) FROM payout_methods WHERE user_id = $1 AND type = $2`, userID, payoutType).Scan(&count)
	return count, err
}

func (r *payoutRepository) Create(ctx context.Context, method *domain.PayoutMethod) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO payout_methods (
			id, user_id, type, is_primary, bank_name, account_number, account_holder_name,
			ewallet_provider, ewallet_phone_number, is_verified, micro_deposit_required, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW(),NOW())
		RETURNING created_at, updated_at
	`,
		method.ID, method.UserID, method.Type, method.IsPrimary, method.BankName, method.AccountNumber, method.AccountHolderName,
		method.EwalletProvider, method.EwalletPhoneNumber, method.IsVerified, method.MicroDepositRequired,
	).Scan(&method.CreatedAt, &method.UpdatedAt)
}

func (r *payoutRepository) Update(ctx context.Context, method *domain.PayoutMethod) error {
	return r.db.QueryRow(ctx, `
		UPDATE payout_methods
		SET bank_name = $1, account_number = $2, account_holder_name = $3, ewallet_provider = $4, ewallet_phone_number = $5,
		    is_primary = $6, is_verified = $7, micro_deposit_required = $8, updated_at = NOW()
		WHERE id = $9 AND user_id = $10
		RETURNING updated_at
	`,
		method.BankName, method.AccountNumber, method.AccountHolderName, method.EwalletProvider, method.EwalletPhoneNumber,
		method.IsPrimary, method.IsVerified, method.MicroDepositRequired, method.ID, method.UserID,
	).Scan(&method.UpdatedAt)
}

func (r *payoutRepository) Delete(ctx context.Context, id, userID domain.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM payout_methods WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *payoutRepository) UnsetPrimaryByUserID(ctx context.Context, userID domain.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE payout_methods SET is_primary = false, updated_at = NOW() WHERE user_id = $1`, userID)
	return err
}

func (r *payoutRepository) SetPrimary(ctx context.Context, id, userID domain.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE payout_methods SET is_primary = false, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE payout_methods SET is_primary = true, updated_at = NOW() WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type cryptoPayoutAddressRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewCryptoPayoutAddressRepository(db *pgxpool.Pool, logger *zap.Logger) domain.CryptoPayoutAddressRepository {
	return &cryptoPayoutAddressRepository{db: db, logger: logger}
}

func (r *cryptoPayoutAddressRepository) ListByUserID(ctx context.Context, userID domain.UUID) ([]*domain.CryptoPayoutAddress, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, network, address, label, created_at
		FROM crypto_payout_addresses
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.CryptoPayoutAddress
	for rows.Next() {
		var a domain.CryptoPayoutAddress
		if err := rows.Scan(&a.ID, &a.UserID, &a.Network, &a.Address, &a.Label, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, nil
}

func (r *cryptoPayoutAddressRepository) GetByIDAndUserID(ctx context.Context, id, userID domain.UUID) (*domain.CryptoPayoutAddress, error) {
	var a domain.CryptoPayoutAddress
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, network, address, label, created_at
		FROM crypto_payout_addresses
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&a.ID, &a.UserID, &a.Network, &a.Address, &a.Label, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *cryptoPayoutAddressRepository) CountByUserID(ctx context.Context, userID domain.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(1) FROM crypto_payout_addresses WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

func (r *cryptoPayoutAddressRepository) Create(ctx context.Context, address *domain.CryptoPayoutAddress) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO crypto_payout_addresses (id, user_id, network, address, label, created_at)
		VALUES ($1,$2,$3,$4,$5,NOW())
		RETURNING created_at
	`, address.ID, address.UserID, address.Network, address.Address, address.Label).Scan(&address.CreatedAt)
}

func (r *cryptoPayoutAddressRepository) Delete(ctx context.Context, id, userID domain.UUID) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM crypto_payout_addresses WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type walletRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewWalletRepository(db *pgxpool.Pool, logger *zap.Logger) domain.WalletRepository {
	return &walletRepository{db: db, logger: logger}
}

func (r *walletRepository) GetByUserID(ctx context.Context, userID domain.UUID) (*domain.Wallet, error) {
	query := `SELECT id, user_id, balance, frozen_balance, currency, updated_at FROM wallets WHERE user_id = $1`
	var w domain.Wallet
	err := r.db.QueryRow(ctx, query, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency, &w.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *walletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	query := `INSERT INTO wallets (id, user_id, balance, frozen_balance, currency, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING updated_at`
	return r.db.QueryRow(ctx, query, wallet.ID, wallet.UserID, wallet.Balance, wallet.FrozenBalance, wallet.Currency).Scan(&wallet.UpdatedAt)
}

func (r *walletRepository) CreditBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	query := `UPDATE wallets SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`
	tag, err := r.db.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *walletRepository) DebitBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	query := `UPDATE wallets SET balance = balance - $1, updated_at = NOW() WHERE user_id = $2 AND balance >= $1`
	tag, err := r.db.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance", nil)
	}
	return nil
}

func (r *walletRepository) FreezeBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	query := `UPDATE wallets SET balance = balance - $1, frozen_balance = frozen_balance + $1, updated_at = NOW()
		WHERE user_id = $2 AND balance >= $1`
	tag, err := r.db.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance to freeze", nil)
	}
	return nil
}

func (r *walletRepository) UnfreezeBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	query := `UPDATE wallets SET frozen_balance = frozen_balance - $1, balance = balance + $1, updated_at = NOW()
		WHERE user_id = $2 AND frozen_balance >= $1`
	tag, err := r.db.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient frozen balance", nil)
	}
	return nil
}

// TransactionRepository
type transactionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewTransactionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.TransactionRepository {
	return &transactionRepository{db: db, logger: logger}
}

func (r *transactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	query := `INSERT INTO transactions (id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, tx.ID, tx.UserID, tx.Type, tx.Amount, tx.Currency, tx.Status, tx.ReferenceID, tx.PaymentMethod, tx.Metadata).Scan(&tx.CreatedAt)
}

func (r *transactionRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Transaction, error) {
	query := `SELECT id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at FROM transactions WHERE id = $1`
	var tx domain.Transaction
	err := r.db.QueryRow(ctx, query, id).Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status, &tx.ReferenceID, &tx.PaymentMethod, &tx.Metadata, &tx.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *transactionRepository) GetByReferenceID(ctx context.Context, refID string) (*domain.Transaction, error) {
	query := `SELECT id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at FROM transactions WHERE reference_id = $1`
	var tx domain.Transaction
	err := r.db.QueryRow(ctx, query, refID).Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status, &tx.ReferenceID, &tx.PaymentMethod, &tx.Metadata, &tx.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &tx, nil
}

func (r *transactionRepository) ListByUser(ctx context.Context, userID domain.UUID, txType string, limit, offset int) ([]*domain.Transaction, error) {
	var query string
	var args []interface{}
	if txType != "" {
		query = `SELECT id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at
			FROM transactions WHERE user_id = $1 AND type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = []interface{}{userID, txType, limit, offset}
	} else {
		query = `SELECT id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at
			FROM transactions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{userID, limit, offset}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status, &tx.ReferenceID, &tx.PaymentMethod, &tx.Metadata, &tx.CreatedAt); err != nil {
			return nil, err
		}
		txs = append(txs, &tx)
	}
	return txs, nil
}

func (r *transactionRepository) UpdateStatus(ctx context.Context, id domain.UUID, status string) error {
	query := `UPDATE transactions SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

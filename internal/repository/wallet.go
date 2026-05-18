package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type txKey struct{}

// WithTx returns a context with the pgx transaction
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// GetTx retrieves the pgx transaction from context
func GetTx(ctx context.Context) pgx.Tx {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return nil
}

type pgxExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type walletRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewWalletRepository(db *pgxpool.Pool, logger *zap.Logger) domain.WalletRepository {
	return &walletRepository{db: db, logger: logger}
}

func (r *walletRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *walletRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	ctxWithTx := WithTx(ctx, tx)
	if err := fn(ctxWithTx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *walletRepository) GetByUserID(ctx context.Context, userID domain.UUID) (*domain.Wallet, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, user_id, balance, frozen_balance, currency, updated_at FROM wallets WHERE user_id = $1`
	var w domain.Wallet
	err := exec.QueryRow(ctx, query, userID).Scan(&w.ID, &w.UserID, &w.Balance, &w.FrozenBalance, &w.Currency, &w.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *walletRepository) Create(ctx context.Context, wallet *domain.Wallet) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO wallets (id, user_id, balance, frozen_balance, currency, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING updated_at`
	return exec.QueryRow(ctx, query, wallet.ID, wallet.UserID, wallet.Balance, wallet.FrozenBalance, wallet.Currency).Scan(&wallet.UpdatedAt)
}

func (r *walletRepository) CreditBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	exec := r.getExecutor(ctx)
	// Execute SELECT FOR UPDATE to ensure row-level locking
	var balance int64
	err := exec.QueryRow(ctx, "SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}

	query := `UPDATE wallets SET balance = balance + $1, updated_at = NOW() WHERE user_id = $2`
	tag, err := exec.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *walletRepository) DebitBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	exec := r.getExecutor(ctx)
	// Execute SELECT FOR UPDATE to ensure row-level locking
	var balance int64
	err := exec.QueryRow(ctx, "SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}

	if balance < amount {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance", nil)
	}

	query := `UPDATE wallets SET balance = balance - $1, updated_at = NOW() WHERE user_id = $2`
	tag, err := exec.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *walletRepository) FreezeBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	exec := r.getExecutor(ctx)
	// Execute SELECT FOR UPDATE to lock row
	var balance int64
	err := exec.QueryRow(ctx, "SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}

	if balance < amount {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance to freeze", nil)
	}

	query := `UPDATE wallets SET balance = balance - $1, frozen_balance = frozen_balance + $1, updated_at = NOW()
		WHERE user_id = $2`
	tag, err := exec.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *walletRepository) UnfreezeBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	exec := r.getExecutor(ctx)
	// Execute SELECT FOR UPDATE to lock row
	var frozenBalance int64
	err := exec.QueryRow(ctx, "SELECT frozen_balance FROM wallets WHERE user_id = $1 FOR UPDATE", userID).Scan(&frozenBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}

	if frozenBalance < amount {
		return domain.NewDomainError(domain.ErrCodeValidation, "insufficient frozen balance", nil)
	}

	query := `UPDATE wallets SET frozen_balance = frozen_balance - $1, balance = balance + $1, updated_at = NOW()
		WHERE user_id = $2`
	tag, err := exec.Exec(ctx, query, amount, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
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

func (r *transactionRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *transactionRepository) Create(ctx context.Context, tx *domain.Transaction) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO transactions (id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()) RETURNING created_at`
	return exec.QueryRow(ctx, query, tx.ID, tx.UserID, tx.Type, tx.Amount, tx.Currency, tx.Status, tx.ReferenceID, tx.PaymentMethod, tx.Metadata).Scan(&tx.CreatedAt)
}

func (r *transactionRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Transaction, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at FROM transactions WHERE id = $1`
	var tx domain.Transaction
	err := exec.QueryRow(ctx, query, id).Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status, &tx.ReferenceID, &tx.PaymentMethod, &tx.Metadata, &tx.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *transactionRepository) GetByReferenceID(ctx context.Context, refID string) (*domain.Transaction, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, user_id, type, amount, currency, status, reference_id, payment_method, metadata, created_at FROM transactions WHERE reference_id = $1`
	var tx domain.Transaction
	err := exec.QueryRow(ctx, query, refID).Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Currency, &tx.Status, &tx.ReferenceID, &tx.PaymentMethod, &tx.Metadata, &tx.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &tx, nil
}

func (r *transactionRepository) ListByUser(ctx context.Context, userID domain.UUID, txType string, limit, offset int) ([]*domain.Transaction, error) {
	exec := r.getExecutor(ctx)
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

	rows, err := exec.Query(ctx, query, args...)
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
	exec := r.getExecutor(ctx)
	query := `UPDATE transactions SET status = $1 WHERE id = $2`
	_, err := exec.Exec(ctx, query, status, id)
	return err
}

package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type creatorTokenRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewCreatorTokenRepository membuat instance baru dari CreatorTokenRepository
func NewCreatorTokenRepository(db *pgxpool.Pool, logger *zap.Logger) domain.CreatorTokenRepository {
	return &creatorTokenRepository{
		db:     db,
		logger: logger,
	}
}

func (r *creatorTokenRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *creatorTokenRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

func (r *creatorTokenRepository) CreateToken(ctx context.Context, token *domain.CreatorToken) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO creator_tokens (id, host_id, name, symbol, total_supply, max_supply, base_price, slope, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at`
	return exec.QueryRow(ctx, query, token.ID, token.HostID, token.Name, token.Symbol, token.TotalSupply, token.MaxSupply, token.BasePrice, token.Slope).Scan(&token.CreatedAt, &token.UpdatedAt)
}

func (r *creatorTokenRepository) GetTokenByHostID(ctx context.Context, hostID domain.UUID) (*domain.CreatorToken, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, host_id, name, symbol, total_supply, max_supply, base_price, slope, created_at, updated_at 
		FROM creator_tokens WHERE host_id = $1`
	var t domain.CreatorToken
	err := exec.QueryRow(ctx, query, hostID).Scan(&t.ID, &t.HostID, &t.Name, &t.Symbol, &t.TotalSupply, &t.MaxSupply, &t.BasePrice, &t.Slope, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *creatorTokenRepository) GetTokenByID(ctx context.Context, id domain.UUID) (*domain.CreatorToken, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, host_id, name, symbol, total_supply, max_supply, base_price, slope, created_at, updated_at 
		FROM creator_tokens WHERE id = $1`
	var t domain.CreatorToken
	err := exec.QueryRow(ctx, query, id).Scan(&t.ID, &t.HostID, &t.Name, &t.Symbol, &t.TotalSupply, &t.MaxSupply, &t.BasePrice, &t.Slope, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *creatorTokenRepository) GetTokenBySymbol(ctx context.Context, symbol string) (*domain.CreatorToken, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, host_id, name, symbol, total_supply, max_supply, base_price, slope, created_at, updated_at 
		FROM creator_tokens WHERE symbol = $1`
	var t domain.CreatorToken
	err := exec.QueryRow(ctx, query, symbol).Scan(&t.ID, &t.HostID, &t.Name, &t.Symbol, &t.TotalSupply, &t.MaxSupply, &t.BasePrice, &t.Slope, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *creatorTokenRepository) GetUserToken(ctx context.Context, userID, tokenID domain.UUID) (*domain.UserToken, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, user_id, token_id, balance, updated_at FROM user_tokens WHERE user_id = $1 AND token_id = $2`
	var ut domain.UserToken
	err := exec.QueryRow(ctx, query, userID, tokenID).Scan(&ut.ID, &ut.UserID, &ut.TokenID, &ut.Balance, &ut.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &ut, nil
}

func (r *creatorTokenRepository) ListUserTokens(ctx context.Context, userID domain.UUID) ([]*domain.UserToken, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT ut.id, ut.user_id, ut.token_id, ut.balance, ut.updated_at,
		ct.id, ct.host_id, ct.name, ct.symbol, ct.total_supply, ct.max_supply, ct.base_price, ct.slope
		FROM user_tokens ut
		JOIN creator_tokens ct ON ut.token_id = ct.id
		WHERE ut.user_id = $1`
	rows, err := exec.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*domain.UserToken
	for rows.Next() {
		var ut domain.UserToken
		var ct domain.CreatorToken
		err := rows.Scan(
			&ut.ID, &ut.UserID, &ut.TokenID, &ut.Balance, &ut.UpdatedAt,
			&ct.ID, &ct.HostID, &ct.Name, &ct.Symbol, &ct.TotalSupply, &ct.MaxSupply, &ct.BasePrice, &ct.Slope,
		)
		if err != nil {
			return nil, err
		}
		ut.Token = &ct
		tokens = append(tokens, &ut)
	}
	return tokens, nil
}

func (r *creatorTokenRepository) UpdateTokenSupply(ctx context.Context, tokenID domain.UUID, newSupply int64) error {
	exec := r.getExecutor(ctx)
	query := `UPDATE creator_tokens SET total_supply = $1, updated_at = NOW() WHERE id = $2`
	tag, err := exec.Exec(ctx, query, newSupply, tokenID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *creatorTokenRepository) UpdateUserTokenBalance(ctx context.Context, userID, tokenID domain.UUID, amount int64) error {
	exec := r.getExecutor(ctx)
	// Check if user token balance record exists
	var utID domain.UUID
	var currentBalance int64
	err := exec.QueryRow(ctx, "SELECT id, balance FROM user_tokens WHERE user_id = $1 AND token_id = $2 FOR UPDATE", userID, tokenID).Scan(&utID, &currentBalance)

	if err != nil {
		if err == pgx.ErrNoRows || errors.Is(err, sql.ErrNoRows) {
			// Insert new record
			newID := domain.NewUUID()
			query := `INSERT INTO user_tokens (id, user_id, token_id, balance, updated_at) VALUES ($1, $2, $3, $4, NOW())`
			_, err = exec.Exec(ctx, query, newID, userID, tokenID, amount)
			return err
		}
		return err
	}

	newBalance := currentBalance + amount
	if newBalance < 0 {
		return domain.NewDomainError(domain.ErrCodeValidation, "saldo token tidak mencukupi", nil)
	}

	query := `UPDATE user_tokens SET balance = $1, updated_at = NOW() WHERE id = $2`
	_, err = exec.Exec(ctx, query, newBalance, utID)
	return err
}

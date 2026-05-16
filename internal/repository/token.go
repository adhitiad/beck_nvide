package repository

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type tokenRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewTokenRepository creates new token repository
func NewTokenRepository(db *pgxpool.Pool, logger *zap.Logger) domain.TokenRepository {
	return &tokenRepository{
		db:     db,
		logger: logger,
	}
}

func (r *tokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, revoked_at)
		VALUES ($1, $2, $3, $4, NOW(), NULL)
		ON CONFLICT (token_hash) DO UPDATE SET
			expires_at = EXCLUDED.expires_at,
			revoked_at = NULL
	`

	_, err := r.db.Exec(ctx, query, token.ID, token.UserID, token.TokenHash, token.ExpiresAt)
	if err != nil {
		r.logger.Error("Failed to create refresh token", zap.Error(err), zap.String("user_id", token.UserID.String()))
		return err
	}
	return nil
}

func (r *tokenRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE id = $1
	`

	token := &domain.RefreshToken{}
	var revokedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, id).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt,
		&token.CreatedAt, &revokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get token by ID", zap.Error(err), zap.String("id", id.String()))
		return nil, err
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

func (r *tokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	token := &domain.RefreshToken{}
	var revokedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt,
		&token.CreatedAt, &revokedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		r.logger.Error("Failed to get token by hash", zap.Error(err))
		return nil, err
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

func (r *tokenRepository) GetActiveByUserID(ctx context.Context, userID domain.UUID) ([]*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]*domain.RefreshToken, 0)
	for rows.Next() {
		token := &domain.RefreshToken{}
		var revokedAt sql.NullTime

		err := rows.Scan(
			&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt,
			&token.CreatedAt, &revokedAt,
		)
		if err != nil {
			return nil, err
		}

		if revokedAt.Valid {
			token.RevokedAt = &revokedAt.Time
		}

		tokens = append(tokens, token)
	}

	return tokens, nil
}

func (r *tokenRepository) RevokeByID(ctx context.Context, id domain.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *tokenRepository) RevokeAllByUserID(ctx context.Context, userID domain.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *tokenRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW()`
	_, err := r.db.Exec(ctx, query)
	return err
}

func (r *tokenRepository) CountActiveByUserID(ctx context.Context, userID domain.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM refresh_tokens
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

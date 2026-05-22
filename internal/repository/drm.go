package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type drmRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewDRMRepository membuat instance baru dari DRMRepository dan memastikan tabel-tabel pendukung siap digunakan
func NewDRMRepository(db *pgxpool.Pool, logger *zap.Logger) domain.DRMRepository {
	repo := &drmRepository{
		db:     db,
		logger: logger,
	}
	repo.ensureTablesExist()
	return repo
}

func (r *drmRepository) ensureTablesExist() {
	ctx := context.Background()
	query := `
	CREATE TABLE IF NOT EXISTS vod_drm_keys (
		vod_id UUID PRIMARY KEY REFERENCES vod_media(id) ON DELETE CASCADE,
		key_value BYTEA NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("Gagal memastikan tabel vod_drm_keys tersedia", zap.Error(err))
	}
}

func (r *drmRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *drmRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

func (r *drmRepository) SaveAccessKey(ctx context.Context, key *domain.VODAccessKey) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO vod_access_keys (id, vod_id, user_id, access_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW()) RETURNING created_at`
	return exec.QueryRow(ctx, query, key.ID, key.VODID, key.UserID, key.AccessToken, key.ExpiresAt).Scan(&key.CreatedAt)
}

func (r *drmRepository) GetAccessKey(ctx context.Context, token string) (*domain.VODAccessKey, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, vod_id, user_id, access_token, expires_at, created_at 
		FROM vod_access_keys WHERE access_token = $1`
	var key domain.VODAccessKey
	err := exec.QueryRow(ctx, query, token).Scan(&key.ID, &key.VODID, &key.UserID, &key.AccessToken, &key.ExpiresAt, &key.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &key, nil
}

func (r *drmRepository) SaveDRMKey(ctx context.Context, vodID domain.UUID, keyValue []byte) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO vod_drm_keys (vod_id, key_value, created_at) 
		VALUES ($1, $2, NOW()) 
		ON CONFLICT (vod_id) DO UPDATE SET key_value = EXCLUDED.key_value`
	_, err := exec.Exec(ctx, query, vodID, keyValue)
	return err
}

func (r *drmRepository) GetDRMKey(ctx context.Context, vodID domain.UUID) ([]byte, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT key_value FROM vod_drm_keys WHERE vod_id = $1`
	var keyValue []byte
	err := exec.QueryRow(ctx, query, vodID).Scan(&keyValue)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return keyValue, nil
}

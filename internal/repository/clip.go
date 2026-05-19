package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type clipRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewClipRepository membuat instance baru dari ClipRepository
func NewClipRepository(db *pgxpool.Pool, logger *zap.Logger) domain.ClipRepository {
	return &clipRepository{
		db:     db,
		logger: logger,
	}
}

func (r *clipRepository) getExecutor(ctx context.Context) pgxExecutor {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *clipRepository) Create(ctx context.Context, clip *domain.StreamClip) error {
	exec := r.getExecutor(ctx)
	query := `INSERT INTO stream_clips (id, stream_id, title, clip_url, duration, score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	return exec.QueryRow(ctx, query, clip.ID, clip.StreamID, clip.Title, clip.ClipURL, clip.Duration, clip.Score).Scan(&clip.CreatedAt)
}

func (r *clipRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.StreamClip, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, stream_id, title, clip_url, duration, score, created_at 
		FROM stream_clips WHERE id = $1`
	var clip domain.StreamClip
	err := exec.QueryRow(ctx, query, id).Scan(&clip.ID, &clip.StreamID, &clip.Title, &clip.ClipURL, &clip.Duration, &clip.Score, &clip.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &clip, nil
}

func (r *clipRepository) ListByStream(ctx context.Context, streamID domain.UUID) ([]*domain.StreamClip, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, stream_id, title, clip_url, duration, score, created_at 
		FROM stream_clips WHERE stream_id = $1 ORDER BY created_at DESC`
	
	rows, err := exec.Query(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.StreamClip
	for rows.Next() {
		var clip domain.StreamClip
		err := rows.Scan(&clip.ID, &clip.StreamID, &clip.Title, &clip.ClipURL, &clip.Duration, &clip.Score, &clip.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &clip)
	}
	return list, nil
}

func (r *clipRepository) ListTrending(ctx context.Context, limit, offset int) ([]*domain.StreamClip, error) {
	exec := r.getExecutor(ctx)
	query := `SELECT id, stream_id, title, clip_url, duration, score, created_at 
		FROM stream_clips ORDER BY score DESC, created_at DESC LIMIT $1 OFFSET $2`
	
	rows, err := exec.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.StreamClip
	for rows.Next() {
		var clip domain.StreamClip
		err := rows.Scan(&clip.ID, &clip.StreamID, &clip.Title, &clip.ClipURL, &clip.Duration, &clip.Score, &clip.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, &clip)
	}
	return list, nil
}

func (r *clipRepository) Delete(ctx context.Context, id domain.UUID) error {
	exec := r.getExecutor(ctx)
	query := `DELETE FROM stream_clips WHERE id = $1`
	_, err := exec.Exec(ctx, query, id)
	return err
}

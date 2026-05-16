package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type streamSessionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewStreamSessionRepository creates a new stream session repository
func NewStreamSessionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.StreamSessionRepository {
	return &streamSessionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *streamSessionRepository) Create(ctx context.Context, session *domain.StreamSession) error {
	query := `
		INSERT INTO stream_sessions (id, stream_id, viewer_id, ip_address, joined_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING joined_at
	`
	err := r.db.QueryRow(ctx, query,
		session.ID,
		session.StreamID,
		session.ViewerID,
		session.IPAddress,
	).Scan(&session.JoinedAt)

	if err != nil {
		r.logger.Error("Failed to create stream session", zap.Error(err))
		return err
	}
	return nil
}

func (r *streamSessionRepository) Update(ctx context.Context, session *domain.StreamSession) error {
	query := `
		UPDATE stream_sessions
		SET left_at = $1, duration = $2
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query,
		session.LeftAt,
		session.Duration,
		session.ID,
	)

	if err != nil {
		r.logger.Error("Failed to update stream session", zap.Error(err))
		return err
	}
	return nil
}

func (r *streamSessionRepository) GetActiveSession(ctx context.Context, streamID, viewerID domain.UUID) (*domain.StreamSession, error) {
	query := `
		SELECT id, stream_id, viewer_id, joined_at, left_at, duration, ip_address
		FROM stream_sessions
		WHERE stream_id = $1 AND viewer_id = $2 AND left_at IS NULL
		ORDER BY joined_at DESC
		LIMIT 1
	`
	var s domain.StreamSession
	err := r.db.QueryRow(ctx, query, streamID, viewerID).Scan(
		&s.ID, &s.StreamID, &s.ViewerID, &s.JoinedAt, &s.LeftAt, &s.Duration, &s.IPAddress,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

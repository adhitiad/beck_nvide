package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type streamRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewStreamRepository creates a new stream repository
func NewStreamRepository(db *pgxpool.Pool, logger *zap.Logger) domain.StreamRepository {
	return &streamRepository{
		db:     db,
		logger: logger,
	}
}

func (r *streamRepository) Create(ctx context.Context, stream *domain.Stream) error {
	query := `
		INSERT INTO streams (id, host_id, title, description, thumbnail_url, status, room_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		stream.ID,
		stream.HostID,
		stream.Title,
		stream.Description,
		stream.ThumbnailURL,
		stream.Status,
		stream.RoomID,
	).Scan(&stream.CreatedAt, &stream.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to create stream", zap.Error(err))
		return err
	}
	return nil
}

func (r *streamRepository) Update(ctx context.Context, stream *domain.Stream) error {
	query := `
		UPDATE streams
		SET title = $1, description = $2, thumbnail_url = $3, status = $4, started_at = $5, ended_at = $6, viewer_peak = $7, total_duration = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		stream.Title,
		stream.Description,
		stream.ThumbnailURL,
		stream.Status,
		stream.StartedAt,
		stream.EndedAt,
		stream.ViewerPeak,
		stream.TotalDuration,
		stream.ID,
	).Scan(&stream.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to update stream", zap.Error(err))
		return err
	}
	return nil
}

func (r *streamRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Stream, error) {
	query := `
		SELECT id, host_id, title, description, thumbnail_url, status, started_at, ended_at, viewer_peak, total_duration, room_id, created_at, updated_at
		FROM streams
		WHERE id = $1
	`
	var s domain.Stream
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.HostID, &s.Title, &s.Description, &s.ThumbnailURL, &s.Status,
		&s.StartedAt, &s.EndedAt, &s.ViewerPeak, &s.TotalDuration, &s.RoomID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *streamRepository) GetByRoomID(ctx context.Context, roomID domain.UUID) (*domain.Stream, error) {
	query := `
		SELECT id, host_id, title, description, thumbnail_url, status, started_at, ended_at, viewer_peak, total_duration, room_id, created_at, updated_at
		FROM streams
		WHERE room_id = $1
	`
	var s domain.Stream
	err := r.db.QueryRow(ctx, query, roomID).Scan(
		&s.ID, &s.HostID, &s.Title, &s.Description, &s.ThumbnailURL, &s.Status,
		&s.StartedAt, &s.EndedAt, &s.ViewerPeak, &s.TotalDuration, &s.RoomID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *streamRepository) GetLiveByHost(ctx context.Context, hostID domain.UUID) (*domain.Stream, error) {
	query := `
		SELECT id, host_id, title, description, thumbnail_url, status, started_at, ended_at, viewer_peak, total_duration, room_id, created_at, updated_at
		FROM streams
		WHERE host_id = $1 AND status = 'live'
	`
	var s domain.Stream
	err := r.db.QueryRow(ctx, query, hostID).Scan(
		&s.ID, &s.HostID, &s.Title, &s.Description, &s.ThumbnailURL, &s.Status,
		&s.StartedAt, &s.EndedAt, &s.ViewerPeak, &s.TotalDuration, &s.RoomID,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *streamRepository) ListLive(ctx context.Context, limit, offset int) ([]*domain.Stream, error) {
	query := `
		SELECT id, host_id, title, description, thumbnail_url, status, started_at, ended_at, viewer_peak, total_duration, room_id, created_at, updated_at
		FROM streams
		WHERE status = 'live'
		ORDER BY started_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []*domain.Stream
	for rows.Next() {
		var s domain.Stream
		err := rows.Scan(
			&s.ID, &s.HostID, &s.Title, &s.Description, &s.ThumbnailURL, &s.Status,
			&s.StartedAt, &s.EndedAt, &s.ViewerPeak, &s.TotalDuration, &s.RoomID,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		streams = append(streams, &s)
	}
	return streams, nil
}

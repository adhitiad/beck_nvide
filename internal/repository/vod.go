package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type vodMediaRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewVODMediaRepository creates a new VOD repository
func NewVODMediaRepository(db *pgxpool.Pool, logger *zap.Logger) domain.VODMediaRepository {
	return &vodMediaRepository{
		db:     db,
		logger: logger,
	}
}

func (r *vodMediaRepository) Create(ctx context.Context, vod *domain.VODMedia) error {
	query := `
		INSERT INTO vod_media (id, user_id, title, description, original_url, hls_url, thumbnail_url, duration, file_size, status, visibility, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		vod.ID,
		vod.UserID,
		vod.Title,
		vod.Description,
		vod.OriginalURL,
		vod.HLSURL,
		vod.ThumbnailURL,
		vod.Duration,
		vod.FileSize,
		vod.Status,
		vod.Visibility,
	).Scan(&vod.CreatedAt, &vod.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to create vod media", zap.Error(err))
		return err
	}
	return nil
}

func (r *vodMediaRepository) Update(ctx context.Context, vod *domain.VODMedia) error {
	query := `
		UPDATE vod_media
		SET title = $1, description = $2, hls_url = $3, thumbnail_url = $4, duration = $5, file_size = $6, status = $7, visibility = $8, updated_at = NOW()
		WHERE id = $9 AND deleted_at IS NULL
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, query,
		vod.Title,
		vod.Description,
		vod.HLSURL,
		vod.ThumbnailURL,
		vod.Duration,
		vod.FileSize,
		vod.Status,
		vod.Visibility,
		vod.ID,
	).Scan(&vod.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to update vod media", zap.Error(err))
		return err
	}
	return nil
}

func (r *vodMediaRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.VODMedia, error) {
	query := `
		SELECT id, user_id, title, description, original_url, hls_url, thumbnail_url, duration, file_size, status, visibility, created_at, updated_at
		FROM vod_media
		WHERE id = $1 AND deleted_at IS NULL
	`
	var v domain.VODMedia
	err := r.db.QueryRow(ctx, query, id).Scan(
		&v.ID, &v.UserID, &v.Title, &v.Description, &v.OriginalURL, &v.HLSURL, &v.ThumbnailURL,
		&v.Duration, &v.FileSize, &v.Status, &v.Visibility, &v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *vodMediaRepository) ListByUser(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.VODMedia, error) {
	query := `
		SELECT id, user_id, title, description, original_url, hls_url, thumbnail_url, duration, file_size, status, visibility, created_at, updated_at
		FROM vod_media
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vods []*domain.VODMedia
	for rows.Next() {
		var v domain.VODMedia
		err := rows.Scan(
			&v.ID, &v.UserID, &v.Title, &v.Description, &v.OriginalURL, &v.HLSURL, &v.ThumbnailURL,
			&v.Duration, &v.FileSize, &v.Status, &v.Visibility, &v.CreatedAt, &v.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		vods = append(vods, &v)
	}
	return vods, nil
}

func (r *vodMediaRepository) Delete(ctx context.Context, id domain.UUID) error {
	query := `UPDATE vod_media SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

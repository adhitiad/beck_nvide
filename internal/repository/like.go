package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type likeRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewLikeRepository creates new like repository
func NewLikeRepository(db *pgxpool.Pool, logger *zap.Logger) domain.LikeRepository {
	return &likeRepository{
		db:     db,
		logger: logger,
	}
}

func (r *likeRepository) Create(ctx context.Context, like *domain.Like) error {
	query := `
		INSERT INTO likes (id, user_id, content_id, content_type, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.Exec(ctx, query, like.ID, like.UserID, like.ContentID, like.ContentType)
	if err != nil {
		r.logger.Error("Failed to create like", zap.Error(err))
		return err
	}
	return nil
}

func (r *likeRepository) Delete(ctx context.Context, userID, contentID domain.UUID, contentType string) error {
	query := `DELETE FROM likes WHERE user_id = $1 AND content_id = $2 AND content_type = $3`
	_, err := r.db.Exec(ctx, query, userID, contentID, contentType)
	return err
}

func (r *likeRepository) HasLiked(ctx context.Context, userID, contentID domain.UUID, contentType string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND content_id = $2 AND content_type = $3)`
	err := r.db.QueryRow(ctx, query, userID, contentID, contentType).Scan(&exists)
	return exists, err
}

func (r *likeRepository) CountByContent(ctx context.Context, contentID domain.UUID, contentType string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM likes WHERE content_id = $1 AND content_type = $2`
	err := r.db.QueryRow(ctx, query, contentID, contentType).Scan(&count)
	return count, err
}

func (r *likeRepository) GetByUser(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Like, error) {
	query := `
		SELECT id, user_id, content_id, content_type, created_at
		FROM likes
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	likes := make([]*domain.Like, 0)
	for rows.Next() {
		like := &domain.Like{}
		err := rows.Scan(&like.ID, &like.UserID, &like.ContentID, &like.ContentType, &like.CreatedAt)
		if err != nil {
			return nil, err
		}
		likes = append(likes, like)
	}
	return likes, nil
}

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type commentRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewCommentRepository creates new comment repository
func NewCommentRepository(db *pgxpool.Pool, logger *zap.Logger) domain.CommentRepository {
	return &commentRepository{
		db:     db,
		logger: logger,
	}
}

func (r *commentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	query := `
		INSERT INTO comments (id, user_id, content_id, content_type, parent_id, content, like_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query,
		comment.ID, comment.UserID, comment.ContentID, comment.ContentType,
		comment.ParentID, comment.Content,
	)
	if err != nil {
		r.logger.Error("Failed to create comment", zap.Error(err))
		return err
	}
	return nil
}

func (r *commentRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, like_count, created_at, updated_at
		FROM comments
		WHERE id = $1
	`
	comment := &domain.Comment{}
	var parentID []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
		&parentID, &comment.Content, &comment.LikeCount, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parentID != nil {
		pid, _ := domain.FromString(string(parentID))
		comment.ParentID = &pid
	}
	return comment, nil
}

func (r *commentRepository) GetByContentID(ctx context.Context, contentID domain.UUID, contentType string, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, like_count, created_at, updated_at
		FROM comments
		WHERE content_id = $1 AND content_type = $2 AND parent_id IS NULL
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, contentID, contentType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0)
	for rows.Next() {
		comment := &domain.Comment{}
		var parentID []byte
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
			&parentID, &comment.Content, &comment.LikeCount, &comment.CreatedAt, &comment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *commentRepository) GetReplies(ctx context.Context, parentID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, like_count, created_at, updated_at
		FROM comments
		WHERE parent_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, parentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0)
	for rows.Next() {
		comment := &domain.Comment{}
		var parentIDVal []byte
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
			&parentIDVal, &comment.Content, &comment.LikeCount, &comment.CreatedAt, &comment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *commentRepository) GetByUser(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, like_count, created_at, updated_at
		FROM comments
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0)
	for rows.Next() {
		comment := &domain.Comment{}
		var parentID []byte
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
			&parentID, &comment.Content, &comment.LikeCount, &comment.CreatedAt, &comment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

func (r *commentRepository) Update(ctx context.Context, comment *domain.Comment) error {
	query := `
		UPDATE comments
		SET content = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, comment.Content, comment.ID)
	return err
}

func (r *commentRepository) Delete(ctx context.Context, id domain.UUID) error {
	query := `DELETE FROM comments WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *commentRepository) IncrementLikeCount(ctx context.Context, id domain.UUID) error {
	query := `UPDATE comments SET like_count = like_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *commentRepository) CountByContent(ctx context.Context, contentID domain.UUID, contentType string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM comments WHERE content_id = $1 AND content_type = $2`
	err := r.db.QueryRow(ctx, query, contentID, contentType).Scan(&count)
	return count, err
}

// Comment Like Repository
type commentLikeRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewCommentLikeRepository creates new comment like repository
func NewCommentLikeRepository(db *pgxpool.Pool, logger *zap.Logger) domain.CommentLikeRepository {
	return &commentLikeRepository{
		db:     db,
		logger: logger,
	}
}

func (r *commentLikeRepository) Create(ctx context.Context, like *domain.CommentLike) error {
	query := `
		INSERT INTO comment_likes (id, user_id, comment_id, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	_, err := r.db.Exec(ctx, query, like.ID, like.UserID, like.CommentID)
	return err
}

func (r *commentLikeRepository) Delete(ctx context.Context, userID, commentID domain.UUID) error {
	query := `DELETE FROM comment_likes WHERE user_id = $1 AND comment_id = $2`
	_, err := r.db.Exec(ctx, query, userID, commentID)
	return err
}

func (r *commentLikeRepository) HasLiked(ctx context.Context, userID, commentID domain.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM comment_likes WHERE user_id = $1 AND comment_id = $2)`
	err := r.db.QueryRow(ctx, query, userID, commentID).Scan(&exists)
	return exists, err
}

func (r *commentLikeRepository) CountByCommentID(ctx context.Context, commentID domain.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM comment_likes WHERE comment_id = $1`
	err := r.db.QueryRow(ctx, query, commentID).Scan(&count)
	return count, err
}

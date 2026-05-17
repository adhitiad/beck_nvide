package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type storyRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewStoryRepository creates new story repository
func NewStoryRepository(db *pgxpool.Pool, logger *zap.Logger) domain.StoryRepository {
	return &storyRepository{
		db:     db,
		logger: logger,
	}
}

func (r *storyRepository) Create(ctx context.Context, story *domain.Story) error {
	query := `
		INSERT INTO stories (id, user_id, content, media_type, expires_at, view_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query, story.ID, story.UserID, story.Content, story.MediaType, story.ExpiresAt)
	if err != nil {
		r.logger.Error("Failed to create story", zap.Error(err), zap.String("user_id", story.UserID.String()))
		return err
	}
	return nil
}

func (r *storyRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Story, error) {
	query := `
		SELECT id, user_id, content, media_type, expires_at, view_count, created_at, updated_at
		FROM stories
		WHERE id = $1
	`
	story := &domain.Story{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&story.ID, &story.UserID, &story.Content, &story.MediaType,
		&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return story, nil
}

func (r *storyRepository) GetByUserID(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Story, error) {
	query := `
		SELECT id, user_id, content, media_type, expires_at, view_count, created_at, updated_at
		FROM stories
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := make([]*domain.Story, 0)
	for rows.Next() {
		story := &domain.Story{}
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Content, &story.MediaType,
			&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	return stories, nil
}

func (r *storyRepository) GetActiveStories(ctx context.Context, userID domain.UUID) ([]*domain.Story, error) {
	query := `
		SELECT s.id, s.user_id, s.content, s.media_type, s.expires_at, s.view_count, s.created_at, s.updated_at
		FROM stories s
		JOIN user_follows f ON s.user_id = f.following_id
		WHERE f.follower_id = $1 AND s.expires_at > NOW()
		ORDER BY s.created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := make([]*domain.Story, 0)
	for rows.Next() {
		story := &domain.Story{}
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Content, &story.MediaType,
			&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	return stories, nil
}

func (r *storyRepository) GetFeedStories(ctx context.Context, userIDs []domain.UUID, limit int) ([]*domain.Story, error) {
	if len(userIDs) == 0 {
		return []*domain.Story{}, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs)+1)
	for i, id := range userIDs {
		placeholders[i] = "$" + string(rune(i+1))
		args[i] = id
	}
	args[len(userIDs)] = limit

	query := `
		SELECT id, user_id, content, media_type, expires_at, view_count, created_at, updated_at
		FROM stories
		WHERE user_id IN (` + placeholders[0] + `) AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(len(userIDs)+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := make([]*domain.Story, 0)
	for rows.Next() {
		story := &domain.Story{}
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Content, &story.MediaType,
			&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stories = append(stories, story)
	}
	return stories, nil
}

func (r *storyRepository) Update(ctx context.Context, story *domain.Story) error {
	query := `
		UPDATE stories
		SET content = $1, media_type = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, story.Content, story.MediaType, story.ID)
	return err
}

func (r *storyRepository) Delete(ctx context.Context, id domain.UUID) error {
	query := `DELETE FROM stories WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *storyRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM stories WHERE expires_at < NOW()`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("Failed to delete expired stories", zap.Error(err))
		return err
	}
	return nil
}

func (r *storyRepository) IncrementViewCount(ctx context.Context, id domain.UUID) error {
	query := `UPDATE stories SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *storyRepository) CountByUser(ctx context.Context, userID domain.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM stories WHERE user_id = $1 AND expires_at > NOW()`
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// Story View Repository
type storyViewRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewStoryViewRepository creates new story view repository
func NewStoryViewRepository(db *pgxpool.Pool, logger *zap.Logger) domain.StoryViewRepository {
	return &storyViewRepository{
		db:     db,
		logger: logger,
	}
}

func (r *storyViewRepository) Create(ctx context.Context, view *domain.StoryView) error {
	query := `
		INSERT INTO story_views (id, story_id, user_id, viewed_at)
		VALUES ($1, $2, $3, NOW())
	`
	_, err := r.db.Exec(ctx, query, view.ID, view.StoryID, view.UserID)
	return err
}

func (r *storyViewRepository) GetByStoryID(ctx context.Context, storyID domain.UUID) ([]*domain.StoryView, error) {
	query := `
		SELECT id, story_id, user_id, viewed_at
		FROM story_views
		WHERE story_id = $1
		ORDER BY viewed_at DESC
	`
	rows, err := r.db.Query(ctx, query, storyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := make([]*domain.StoryView, 0)
	for rows.Next() {
		view := &domain.StoryView{}
		err := rows.Scan(&view.ID, &view.StoryID, &view.UserID, &view.ViewedAt)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (r *storyViewRepository) GetByUserIDAndStoryID(ctx context.Context, userID, storyID domain.UUID) (*domain.StoryView, error) {
	query := `
		SELECT id, story_id, user_id, viewed_at
		FROM story_views
		WHERE user_id = $1 AND story_id = $2
	`
	view := &domain.StoryView{}
	err := r.db.QueryRow(ctx, query, userID, storyID).Scan(
		&view.ID, &view.StoryID, &view.UserID, &view.ViewedAt,
	)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func (r *storyViewRepository) HasViewed(ctx context.Context, userID, storyID domain.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM story_views WHERE user_id = $1 AND story_id = $2)`
	err := r.db.QueryRow(ctx, query, userID, storyID).Scan(&exists)
	return exists, err
}

func (r *storyViewRepository) CountByStoryID(ctx context.Context, storyID domain.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM story_views WHERE story_id = $1`
	err := r.db.QueryRow(ctx, query, storyID).Scan(&count)
	return count, err
}
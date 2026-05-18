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
		INSERT INTO stories (id, user_id, content, media_type, expires_at, view_count, created_at, updated_at, media_url, caption, is_expired)
		VALUES ($1, $2, $3, $4, $5, 0, NOW(), NOW(), $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query, 
		story.ID, story.UserID, story.Content, story.MediaType, story.ExpiresAt,
		story.MediaURL, story.Caption, story.IsExpired,
	)
	if err != nil {
		r.logger.Error("Failed to create story", zap.Error(err), zap.String("user_id", story.UserID.String()))
		return err
	}
	return nil
}

func (r *storyRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Story, error) {
	query := `
		SELECT s.id, s.user_id, s.content, s.media_type, s.expires_at, s.view_count, s.created_at, s.updated_at, s.media_url, s.caption, s.is_expired,
		       u.id, u.username, u.email, u.role_id, u.avatar_url
		FROM stories s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = $1
	`
	story := &domain.Story{}
	user := &domain.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&story.ID, &story.UserID, &story.Content, &story.MediaType,
		&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
		&story.MediaURL, &story.Caption, &story.IsExpired,
		&user.ID, &user.Username, &user.Email, &user.RoleID, &user.AvatarURL,
	)
	if err != nil {
		return nil, err
	}
	story.User = user
	return story, nil
}

func (r *storyRepository) GetByUserID(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Story, error) {
	query := `
		SELECT s.id, s.user_id, s.content, s.media_type, s.expires_at, s.view_count, s.created_at, s.updated_at, s.media_url, s.caption, s.is_expired,
		       u.id, u.username, u.email, u.role_id, u.avatar_url
		FROM stories s
		JOIN users u ON s.user_id = u.id
		WHERE s.user_id = $1 AND s.expires_at > NOW() AND s.is_expired = false
		ORDER BY s.created_at DESC
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
		user := &domain.User{}
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Content, &story.MediaType,
			&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
			&story.MediaURL, &story.Caption, &story.IsExpired,
			&user.ID, &user.Username, &user.Email, &user.RoleID, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		story.User = user
		stories = append(stories, story)
	}
	return stories, nil
}

func (r *storyRepository) GetActiveStories(ctx context.Context, userID domain.UUID) ([]*domain.Story, error) {
	query := `
		SELECT s.id, s.user_id, s.content, s.media_type, s.expires_at, s.view_count, s.created_at, s.updated_at, s.media_url, s.caption, s.is_expired,
		       u.id, u.username, u.email, u.role_id, u.avatar_url
		FROM stories s
		JOIN user_follows f ON s.user_id = f.following_id
		JOIN users u ON s.user_id = u.id
		WHERE f.follower_id = $1 AND s.expires_at > NOW() AND s.is_expired = false
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
		user := &domain.User{}
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Content, &story.MediaType,
			&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
			&story.MediaURL, &story.Caption, &story.IsExpired,
			&user.ID, &user.Username, &user.Email, &user.RoleID, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		story.User = user
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
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}
	args[len(userIDs)] = limit

	query := `
		SELECT s.id, s.user_id, s.content, s.media_type, s.expires_at, s.view_count, s.created_at, s.updated_at, s.media_url, s.caption, s.is_expired,
		       u.id, u.username, u.email, u.role_id, u.avatar_url
		FROM stories s
		JOIN users u ON s.user_id = u.id
		WHERE s.user_id IN (` + makeINPlaceholders(len(userIDs)) + `) AND s.expires_at > NOW() AND s.is_expired = false
		ORDER BY s.created_at DESC
		LIMIT $` + strconv.Itoa(len(userIDs)+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := make([]*domain.Story, 0)
	for rows.Next() {
		story := &domain.Story{}
		user := &domain.User{}
		err := rows.Scan(
			&story.ID, &story.UserID, &story.Content, &story.MediaType,
			&story.ExpiresAt, &story.ViewCount, &story.CreatedAt, &story.UpdatedAt,
			&story.MediaURL, &story.Caption, &story.IsExpired,
			&user.ID, &user.Username, &user.Email, &user.RoleID, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		story.User = user
		stories = append(stories, story)
	}
	return stories, nil
}

func makeINPlaceholders(n int) string {
	var s string
	for i := 1; i <= n; i++ {
		if i > 1 {
			s += ", "
		}
		s += "$" + strconv.Itoa(i)
	}
	return s
}

func (r *storyRepository) Update(ctx context.Context, story *domain.Story) error {
	query := `
		UPDATE stories
		SET content = $1, media_type = $2, media_url = $3, caption = $4, is_expired = $5, updated_at = NOW()
		WHERE id = $6
	`
	_, err := r.db.Exec(ctx, query, story.Content, story.MediaType, story.MediaURL, story.Caption, story.IsExpired, story.ID)
	return err
}

func (r *storyRepository) Delete(ctx context.Context, id domain.UUID) error {
	query := `DELETE FROM stories WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *storyRepository) DeleteExpired(ctx context.Context) error {
	query := `UPDATE stories SET is_expired = true WHERE expires_at < NOW() AND is_expired = false`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("Failed to update expired stories to is_expired = true", zap.Error(err))
		return err
	}
	return nil
}

func (r *storyRepository) IncrementViewCount(ctx context.Context, id domain.UUID) error {
	query := `UPDATE stories SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *storyRepository) AddViewCount(ctx context.Context, id domain.UUID, delta int) error {
	query := `UPDATE stories SET view_count = view_count + $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, delta, id)
	return err
}

func (r *storyRepository) CountByUser(ctx context.Context, userID domain.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM stories WHERE user_id = $1 AND expires_at > NOW() AND is_expired = false`
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
		INSERT INTO story_views (id, story_id, viewer_id, viewed_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (story_id, viewer_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, view.ID, view.StoryID, view.ViewerID)
	return err
}

func (r *storyViewRepository) GetByStoryID(ctx context.Context, storyID domain.UUID) ([]*domain.StoryView, error) {
	query := `
		SELECT id, story_id, viewer_id, viewed_at
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
		err := rows.Scan(&view.ID, &view.StoryID, &view.ViewerID, &view.ViewedAt)
		if err != nil {
			return nil, err
		}
		// Populate legacy UserID for complete safety
		view.UserID = view.ViewerID
		views = append(views, view)
	}
	return views, nil
}

func (r *storyViewRepository) GetByUserIDAndStoryID(ctx context.Context, userID, storyID domain.UUID) (*domain.StoryView, error) {
	query := `
		SELECT id, story_id, viewer_id, viewed_at
		FROM story_views
		WHERE viewer_id = $1 AND story_id = $2
	`
	view := &domain.StoryView{}
	err := r.db.QueryRow(ctx, query, userID, storyID).Scan(
		&view.ID, &view.StoryID, &view.ViewerID, &view.ViewedAt,
	)
	if err != nil {
		return nil, err
	}
	view.UserID = view.ViewerID
	return view, nil
}

func (r *storyViewRepository) HasViewed(ctx context.Context, userID, storyID domain.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM story_views WHERE viewer_id = $1 AND story_id = $2)`
	err := r.db.QueryRow(ctx, query, userID, storyID).Scan(&exists)
	return exists, err
}

func (r *storyViewRepository) CountByStoryID(ctx context.Context, storyID domain.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM story_views WHERE story_id = $1`
	err := r.db.QueryRow(ctx, query, storyID).Scan(&count)
	return count, err
}
package repository

import (
	"context"
	"strconv"

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
		INSERT INTO comments (id, user_id, content_id, content_type, parent_id, content, text, like_count, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, false, NOW(), NOW())
	`
	// Align text and content fields
	if comment.Text == "" {
		comment.Text = comment.Content
	}
	if comment.Content == "" {
		comment.Content = comment.Text
	}

	_, err := r.db.Exec(ctx, query,
		comment.ID, comment.UserID, comment.ContentID, comment.ContentType,
		comment.ParentID, comment.Content, comment.Text,
	)
	if err != nil {
		r.logger.Error("Failed to create comment", zap.Error(err))
		return err
	}
	return nil
}

func (r *commentRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, text, like_count, is_deleted, created_at, updated_at
		FROM comments
		WHERE id = $1 AND is_deleted = false
	`
	comment := &domain.Comment{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
		&comment.ParentID, &comment.Content, &comment.Text, &comment.LikeCount,
		&comment.IsDeleted, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch single user to populate User relation
	userQuery := `SELECT id, username, email, role_id, avatar_url FROM users WHERE id = $1`
	user := &domain.User{}
	err = r.db.QueryRow(ctx, userQuery, comment.UserID).Scan(
		&user.ID, &user.Username, &user.Email, &user.RoleID, &user.AvatarURL,
	)
	if err == nil {
		comment.User = user
	}

	return comment, nil
}

func (r *commentRepository) GetByContentID(ctx context.Context, contentID domain.UUID, contentType string, limit, offset int) ([]*domain.Comment, error) {
	// LATERAL JOIN to fetch nested replies (1 level max) in ONE single query!
	query := `
		SELECT 
			c.id, c.user_id, c.content_id, c.content_type, c.parent_id, c.content, c.text, c.like_count, c.is_deleted, c.created_at, c.updated_at,
			r.id, r.user_id, r.content_id, r.content_type, r.parent_id, r.content, r.text, r.like_count, r.is_deleted, r.created_at, r.updated_at
		FROM comments c
		LEFT JOIN LATERAL (
			SELECT id, user_id, content_id, content_type, parent_id, content, text, like_count, is_deleted, created_at, updated_at
			FROM comments
			WHERE parent_id = c.id AND is_deleted = false
			ORDER BY created_at ASC
			LIMIT 10
		) r ON true
		WHERE c.content_id = $1 AND c.content_type = $2 AND c.parent_id IS NULL AND c.is_deleted = false
		ORDER BY c.created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, query, contentID, contentType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commentsMap := make(map[domain.UUID]*domain.Comment)
	commentsList := make([]*domain.Comment, 0)
	userIdsMap := make(map[domain.UUID]bool)

	for rows.Next() {
		c := &domain.Comment{}
		rep := &domain.Comment{}
		
		var repID, repUserID, repContentID, repParentID *domain.UUID
		var repContentType, repContent, repText *string
		var repLikeCount *int
		var repIsDeleted *bool
		var repCreatedAt, repUpdatedAt *string

		err := rows.Scan(
			&c.ID, &c.UserID, &c.ContentID, &c.ContentType, &c.ParentID, &c.Content, &c.Text, &c.LikeCount, &c.IsDeleted, &c.CreatedAt, &c.UpdatedAt,
			&repID, &repUserID, &repContentID, &repParentID, &repContentType, &repContent, &repText, &repLikeCount, &repIsDeleted, &repCreatedAt, &repUpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		userIdsMap[c.UserID] = true

		parent, exists := commentsMap[c.ID]
		if !exists {
			parent = c
			parent.Replies = make([]*domain.Comment, 0)
			commentsMap[c.ID] = parent
			commentsList = append(commentsList, parent)
		}

		// If there is a reply in the lateral join row
		if repID != nil {
			rep.ID = *repID
			rep.UserID = *repUserID
			rep.ContentID = *repContentID
			rep.ContentType = *repContentType
			rep.ParentID = repParentID
			rep.Content = *repContent
			rep.Text = *repText
			rep.LikeCount = *repLikeCount
			rep.IsDeleted = *repIsDeleted
			
			userIdsMap[rep.UserID] = true
			
			// Deduplicate replies inside the list
			isDuplicate := false
			for _, existingRep := range parent.Replies {
				if existingRep.ID == rep.ID {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				parent.Replies = append(parent.Replies, rep)
			}
		}
	}

	// N+1 Query Prevention: Batch fetch user info using IN query
	if len(userIdsMap) > 0 {
		userIds := make([]domain.UUID, 0, len(userIdsMap))
		for uid := range userIdsMap {
			userIds = append(userIds, uid)
		}

		placeholders := make([]string, len(userIds))
		args := make([]interface{}, len(userIds))
		for i, uid := range userIds {
			placeholders[i] = "$" + strconv.Itoa(i+1)
			args[i] = uid
		}

		userQuery := "SELECT id, username, email, role_id, avatar_url FROM users WHERE id IN (" + makeINPlaceholders(len(userIds)) + ")"
		userRows, err := r.db.Query(ctx, userQuery, args...)
		if err != nil {
			r.logger.Warn("Failed to batch fetch users for comments", zap.Error(err))
		} else {
			defer userRows.Close()
			usersMap := make(map[domain.UUID]*domain.User)
			for userRows.Next() {
				u := &domain.User{}
				if err := userRows.Scan(&u.ID, &u.Username, &u.Email, &u.RoleID, &u.AvatarURL); err == nil {
					usersMap[u.ID] = u
				}
			}

			// Map users back to comments and replies
			for _, comment := range commentsList {
				comment.User = usersMap[comment.UserID]
				for _, reply := range comment.Replies {
					reply.User = usersMap[reply.UserID]
				}
			}
		}
	}

	return commentsList, nil
}

func (r *commentRepository) GetReplies(ctx context.Context, parentID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, text, like_count, is_deleted, created_at, updated_at
		FROM comments
		WHERE parent_id = $1 AND is_deleted = false
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, parentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*domain.Comment, 0)
	userIdsMap := make(map[domain.UUID]bool)

	for rows.Next() {
		comment := &domain.Comment{}
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
			&comment.ParentID, &comment.Content, &comment.Text, &comment.LikeCount,
			&comment.IsDeleted, &comment.CreatedAt, &comment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		userIdsMap[comment.UserID] = true
		comments = append(comments, comment)
	}

	// Batch fetch users
	if len(userIdsMap) > 0 {
		userIds := make([]domain.UUID, 0, len(userIdsMap))
		for uid := range userIdsMap {
			userIds = append(userIds, uid)
		}

		args := make([]interface{}, len(userIds))
		for i, uid := range userIds {
			args[i] = uid
		}

		userQuery := "SELECT id, username, email, role_id, avatar_url FROM users WHERE id IN (" + makeINPlaceholders(len(userIds)) + ")"
		userRows, err := r.db.Query(ctx, userQuery, args...)
		if err == nil {
			defer userRows.Close()
			usersMap := make(map[domain.UUID]*domain.User)
			for userRows.Next() {
				u := &domain.User{}
				if err := userRows.Scan(&u.ID, &u.Username, &u.Email, &u.RoleID, &u.AvatarURL); err == nil {
					usersMap[u.ID] = u
				}
			}
			for _, comment := range comments {
				comment.User = usersMap[comment.UserID]
			}
		}
	}

	return comments, nil
}

func (r *commentRepository) GetByUser(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Comment, error) {
	query := `
		SELECT id, user_id, content_id, content_type, parent_id, content, text, like_count, is_deleted, created_at, updated_at
		FROM comments
		WHERE user_id = $1 AND is_deleted = false
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
		err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ContentID, &comment.ContentType,
			&comment.ParentID, &comment.Content, &comment.Text, &comment.LikeCount,
			&comment.IsDeleted, &comment.CreatedAt, &comment.UpdatedAt,
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
		SET content = $1, text = $2, is_deleted = $3, updated_at = NOW()
		WHERE id = $4
	`
	if comment.Text == "" {
		comment.Text = comment.Content
	}
	if comment.Content == "" {
		comment.Content = comment.Text
	}
	_, err := r.db.Exec(ctx, query, comment.Content, comment.Text, comment.IsDeleted, comment.ID)
	return err
}

func (r *commentRepository) Delete(ctx context.Context, id domain.UUID) error {
	// Soft delete comment
	query := `UPDATE comments SET is_deleted = true, updated_at = NOW() WHERE id = $1`
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
	query := `SELECT COUNT(*) FROM comments WHERE content_id = $1 AND content_type = $2 AND is_deleted = false`
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
		ON CONFLICT (user_id, comment_id) DO NOTHING
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

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type shortVideoRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewShortVideoRepository(db *pgxpool.Pool, logger *zap.Logger) domain.ShortVideoRepository {
	return &shortVideoRepository{db: db, logger: logger}
}

func (r *shortVideoRepository) Create(ctx context.Context, video *domain.ShortVideo) error {
	query := `INSERT INTO short_videos (id, user_id, video_url, thumbnail_url, caption, duration,
		like_count, comment_count, share_count, view_count, gift_value, status, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, 0, 0, 0, 0, $7, $8, NOW(), NOW()) RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query, video.ID, video.UserID, video.VideoURL, video.ThumbnailURL,
		video.Caption, video.Duration, video.Status, video.Tags).Scan(&video.CreatedAt, &video.UpdatedAt)
}

func (r *shortVideoRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.ShortVideo, error) {
	query := `SELECT sv.id, sv.user_id, sv.video_url, sv.thumbnail_url, sv.caption, sv.duration,
		sv.like_count, sv.comment_count, sv.share_count, sv.view_count, sv.gift_value,
		sv.status, sv.tags, sv.created_at, sv.updated_at,
		u.id, u.username, u.avatar_url
		FROM short_videos sv JOIN users u ON sv.user_id = u.id WHERE sv.id = $1`
	var v domain.ShortVideo
	var u domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(&v.ID, &v.UserID, &v.VideoURL, &v.ThumbnailURL,
		&v.Caption, &v.Duration, &v.LikeCount, &v.CommentCount, &v.ShareCount, &v.ViewCount,
		&v.GiftValue, &v.Status, &v.Tags, &v.CreatedAt, &v.UpdatedAt,
		&u.ID, &u.Username, &u.AvatarURL)
	if err != nil {
		return nil, err
	}
	v.User = &u
	return &v, nil
}

func (r *shortVideoRepository) Update(ctx context.Context, video *domain.ShortVideo) error {
	query := `UPDATE short_videos SET caption=$1, status=$2, tags=$3, updated_at=NOW() WHERE id=$4`
	_, err := r.db.Exec(ctx, query, video.Caption, video.Status, video.Tags, video.ID)
	return err
}

func (r *shortVideoRepository) Delete(ctx context.Context, id domain.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE short_videos SET status='deleted', updated_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *shortVideoRepository) GetFeed(ctx context.Context, viewerID domain.UUID, limit, offset int) ([]*domain.ShortVideo, error) {
	query := `SELECT sv.id, sv.user_id, sv.video_url, sv.thumbnail_url, sv.caption, sv.duration,
		sv.like_count, sv.comment_count, sv.share_count, sv.view_count, sv.gift_value,
		sv.status, sv.tags, sv.created_at, sv.updated_at,
		u.id, u.username, u.avatar_url,
		EXISTS(SELECT 1 FROM short_video_likes WHERE video_id=sv.id AND user_id=$1) as is_liked
		FROM short_videos sv JOIN users u ON sv.user_id = u.id
		WHERE sv.status = 'active'
		ORDER BY sv.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, viewerID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ShortVideo
	for rows.Next() {
		var v domain.ShortVideo
		var u domain.User
		if err := rows.Scan(&v.ID, &v.UserID, &v.VideoURL, &v.ThumbnailURL, &v.Caption, &v.Duration,
			&v.LikeCount, &v.CommentCount, &v.ShareCount, &v.ViewCount, &v.GiftValue,
			&v.Status, &v.Tags, &v.CreatedAt, &v.UpdatedAt,
			&u.ID, &u.Username, &u.AvatarURL, &v.IsLiked); err != nil {
			return nil, err
		}
		v.User = &u
		list = append(list, &v)
	}
	return list, nil
}

func (r *shortVideoRepository) GetByUserID(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.ShortVideo, error) {
	query := `SELECT id, user_id, video_url, thumbnail_url, caption, duration,
		like_count, comment_count, share_count, view_count, gift_value,
		status, tags, created_at, updated_at
		FROM short_videos WHERE user_id=$1 AND status != 'deleted'
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ShortVideo
	for rows.Next() {
		var v domain.ShortVideo
		if err := rows.Scan(&v.ID, &v.UserID, &v.VideoURL, &v.ThumbnailURL, &v.Caption, &v.Duration,
			&v.LikeCount, &v.CommentCount, &v.ShareCount, &v.ViewCount, &v.GiftValue,
			&v.Status, &v.Tags, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, &v)
	}
	return list, nil
}

func (r *shortVideoRepository) GetTrending(ctx context.Context, limit, offset int) ([]*domain.ShortVideo, error) {
	// Trending score = likes*3 + comments*2 + shares*5 + views*0.1 + gifts*0.01
	query := `SELECT sv.id, sv.user_id, sv.video_url, sv.thumbnail_url, sv.caption, sv.duration,
		sv.like_count, sv.comment_count, sv.share_count, sv.view_count, sv.gift_value,
		sv.status, sv.tags, sv.created_at, sv.updated_at,
		u.id, u.username, u.avatar_url
		FROM short_videos sv JOIN users u ON sv.user_id = u.id
		WHERE sv.status = 'active' AND sv.created_at > NOW() - INTERVAL '7 days'
		ORDER BY (sv.like_count*3 + sv.comment_count*2 + sv.share_count*5 + sv.view_count*0.1 + sv.gift_value*0.01) DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ShortVideo
	for rows.Next() {
		var v domain.ShortVideo
		var u domain.User
		if err := rows.Scan(&v.ID, &v.UserID, &v.VideoURL, &v.ThumbnailURL, &v.Caption, &v.Duration,
			&v.LikeCount, &v.CommentCount, &v.ShareCount, &v.ViewCount, &v.GiftValue,
			&v.Status, &v.Tags, &v.CreatedAt, &v.UpdatedAt,
			&u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		v.User = &u
		list = append(list, &v)
	}
	return list, nil
}

func (r *shortVideoRepository) IncrementViewCount(ctx context.Context, id domain.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE short_videos SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

func (r *shortVideoRepository) IncrementShareCount(ctx context.Context, id domain.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE short_videos SET share_count = share_count + 1 WHERE id = $1`, id)
	return err
}

func (r *shortVideoRepository) UpdateGiftValue(ctx context.Context, id domain.UUID, amount int64) error {
	_, err := r.db.Exec(ctx, `UPDATE short_videos SET gift_value = gift_value + $1 WHERE id = $2`, amount, id)
	return err
}

func (r *shortVideoRepository) Like(ctx context.Context, like *domain.ShortVideoLike) error {
	query := `INSERT INTO short_video_likes (id, video_id, user_id, created_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (video_id, user_id) DO NOTHING`
	_, err := r.db.Exec(ctx, query, like.ID, like.VideoID, like.UserID)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE short_videos SET like_count = like_count + 1 WHERE id = $1`, like.VideoID)
	return err
}

func (r *shortVideoRepository) Unlike(ctx context.Context, videoID, userID domain.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM short_video_likes WHERE video_id=$1 AND user_id=$2`, videoID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		_, err = r.db.Exec(ctx, `UPDATE short_videos SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, videoID)
	}
	return err
}

func (r *shortVideoRepository) HasLiked(ctx context.Context, videoID, userID domain.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM short_video_likes WHERE video_id=$1 AND user_id=$2)`,
		videoID, userID).Scan(&exists)
	return exists, err
}

func (r *shortVideoRepository) CreateComment(ctx context.Context, comment *domain.ShortVideoComment) error {
	query := `INSERT INTO short_video_comments (id, video_id, user_id, content, parent_id, like_count, created_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW()) RETURNING created_at`
	err := r.db.QueryRow(ctx, query, comment.ID, comment.VideoID, comment.UserID,
		comment.Content, comment.ParentID).Scan(&comment.CreatedAt)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE short_videos SET comment_count = comment_count + 1 WHERE id = $1`, comment.VideoID)
	return err
}

func (r *shortVideoRepository) GetComments(ctx context.Context, videoID domain.UUID, limit, offset int) ([]*domain.ShortVideoComment, error) {
	query := `SELECT c.id, c.video_id, c.user_id, c.content, c.parent_id, c.like_count, c.created_at,
		u.id, u.username, u.avatar_url
		FROM short_video_comments c JOIN users u ON c.user_id = u.id
		WHERE c.video_id=$1 AND c.parent_id IS NULL
		ORDER BY c.created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, videoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ShortVideoComment
	for rows.Next() {
		var c domain.ShortVideoComment
		var u domain.User
		if err := rows.Scan(&c.ID, &c.VideoID, &c.UserID, &c.Content, &c.ParentID, &c.LikeCount,
			&c.CreatedAt, &u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		c.User = &u
		list = append(list, &c)
	}
	return list, nil
}

func (r *shortVideoRepository) GetCommentReplies(ctx context.Context, parentID domain.UUID, limit, offset int) ([]*domain.ShortVideoComment, error) {
	query := `SELECT c.id, c.video_id, c.user_id, c.content, c.parent_id, c.like_count, c.created_at,
		u.id, u.username, u.avatar_url
		FROM short_video_comments c JOIN users u ON c.user_id = u.id
		WHERE c.parent_id=$1 ORDER BY c.created_at ASC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, parentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.ShortVideoComment
	for rows.Next() {
		var c domain.ShortVideoComment
		var u domain.User
		if err := rows.Scan(&c.ID, &c.VideoID, &c.UserID, &c.Content, &c.ParentID, &c.LikeCount,
			&c.CreatedAt, &u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		c.User = &u
		list = append(list, &c)
	}
	return list, nil
}

func (r *shortVideoRepository) DeleteComment(ctx context.Context, id domain.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM short_video_comments WHERE id=$1`, id)
	return err
}

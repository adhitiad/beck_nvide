package repository

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type streamTagRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewStreamTagRepository(db *pgxpool.Pool, logger *zap.Logger) domain.StreamTagRepository {
	return &streamTagRepository{db: db, logger: logger}
}

func (r *streamTagRepository) Create(ctx context.Context, tag *domain.StreamTag) error {
	query := `INSERT INTO stream_tags (id, name, category, keywords, created_at)
		VALUES ($1, $2, $3, $4, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, tag.ID, tag.Name, tag.Category, tag.Keywords).Scan(&tag.CreatedAt)
}

func (r *streamTagRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.StreamTag, error) {
	query := `SELECT id, name, category, keywords, created_at FROM stream_tags WHERE id=$1`
	var tag domain.StreamTag
	err := r.db.QueryRow(ctx, query, id).Scan(&tag.ID, &tag.Name, &tag.Category, &tag.Keywords, &tag.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *streamTagRepository) GetByName(ctx context.Context, name string) (*domain.StreamTag, error) {
	query := `SELECT id, name, category, keywords, created_at FROM stream_tags WHERE name=$1`
	var tag domain.StreamTag
	err := r.db.QueryRow(ctx, query, strings.ToLower(name)).Scan(&tag.ID, &tag.Name, &tag.Category,
		&tag.Keywords, &tag.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *streamTagRepository) ListAll(ctx context.Context) ([]*domain.StreamTag, error) {
	query := `SELECT id, name, category, keywords, created_at FROM stream_tags ORDER BY category, name`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.StreamTag
	for rows.Next() {
		var tag domain.StreamTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.Keywords, &tag.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &tag)
	}
	return list, nil
}

func (r *streamTagRepository) ListByCategory(ctx context.Context, category string) ([]*domain.StreamTag, error) {
	query := `SELECT id, name, category, keywords, created_at FROM stream_tags WHERE category=$1 ORDER BY name`
	rows, err := r.db.Query(ctx, query, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.StreamTag
	for rows.Next() {
		var tag domain.StreamTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.Keywords, &tag.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &tag)
	}
	return list, nil
}

func (r *streamTagRepository) AddTagToStream(ctx context.Context, streamID, tagID domain.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO stream_tag_map (stream_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		streamID, tagID)
	return err
}

func (r *streamTagRepository) RemoveTagFromStream(ctx context.Context, streamID, tagID domain.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM stream_tag_map WHERE stream_id=$1 AND tag_id=$2`, streamID, tagID)
	return err
}

func (r *streamTagRepository) GetStreamTags(ctx context.Context, streamID domain.UUID) ([]*domain.StreamTag, error) {
	query := `SELECT st.id, st.name, st.category, st.keywords, st.created_at
		FROM stream_tags st JOIN stream_tag_map stm ON st.id = stm.tag_id
		WHERE stm.stream_id=$1 ORDER BY st.name`
	rows, err := r.db.Query(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.StreamTag
	for rows.Next() {
		var tag domain.StreamTag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Category, &tag.Keywords, &tag.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &tag)
	}
	return list, nil
}

func (r *streamTagRepository) ClearStreamTags(ctx context.Context, streamID domain.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM stream_tag_map WHERE stream_id=$1`, streamID)
	return err
}

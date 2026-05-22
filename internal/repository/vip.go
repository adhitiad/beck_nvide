package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type vipRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewVIPRepository(db *pgxpool.Pool, logger *zap.Logger) domain.VIPRepository {
	return &vipRepository{db: db, logger: logger}
}

func (r *vipRepository) ListLevels(ctx context.Context) ([]*domain.VIPLevel, error) {
	query := `SELECT id, name, display_name, price, duration_days, badge_url, chat_color,
		name_glow_color, privileges, sort_order, created_at
		FROM vip_levels ORDER BY sort_order ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var levels []*domain.VIPLevel
	for rows.Next() {
		var l domain.VIPLevel
		if err := rows.Scan(&l.ID, &l.Name, &l.DisplayName, &l.Price, &l.DurationDays,
			&l.BadgeURL, &l.ChatColor, &l.NameGlowColor, &l.Privileges, &l.SortOrder, &l.CreatedAt); err != nil {
			return nil, err
		}
		levels = append(levels, &l)
	}
	return levels, nil
}

func (r *vipRepository) GetLevelByID(ctx context.Context, id domain.UUID) (*domain.VIPLevel, error) {
	query := `SELECT id, name, display_name, price, duration_days, badge_url, chat_color,
		name_glow_color, privileges, sort_order, created_at
		FROM vip_levels WHERE id = $1`
	var l domain.VIPLevel
	err := r.db.QueryRow(ctx, query, id).Scan(&l.ID, &l.Name, &l.DisplayName, &l.Price,
		&l.DurationDays, &l.BadgeURL, &l.ChatColor, &l.NameGlowColor, &l.Privileges,
		&l.SortOrder, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *vipRepository) GetLevelByName(ctx context.Context, name string) (*domain.VIPLevel, error) {
	query := `SELECT id, name, display_name, price, duration_days, badge_url, chat_color,
		name_glow_color, privileges, sort_order, created_at
		FROM vip_levels WHERE name = $1`
	var l domain.VIPLevel
	err := r.db.QueryRow(ctx, query, name).Scan(&l.ID, &l.Name, &l.DisplayName, &l.Price,
		&l.DurationDays, &l.BadgeURL, &l.ChatColor, &l.NameGlowColor, &l.Privileges,
		&l.SortOrder, &l.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *vipRepository) Subscribe(ctx context.Context, uv *domain.UserVIP) error {
	query := `INSERT INTO user_vip (id, user_id, vip_level_id, started_at, expires_at, auto_renew, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, uv.ID, uv.UserID, uv.VIPLevelID, uv.StartedAt,
		uv.ExpiresAt, uv.AutoRenew).Scan(&uv.CreatedAt)
}

func (r *vipRepository) GetActiveByUserID(ctx context.Context, userID domain.UUID) (*domain.UserVIP, error) {
	query := `SELECT uv.id, uv.user_id, uv.vip_level_id, uv.started_at, uv.expires_at,
		uv.auto_renew, uv.created_at,
		vl.id, vl.name, vl.display_name, vl.price, vl.duration_days, vl.badge_url,
		vl.chat_color, vl.name_glow_color, vl.privileges, vl.sort_order, vl.created_at
		FROM user_vip uv
		JOIN vip_levels vl ON uv.vip_level_id = vl.id
		WHERE uv.user_id = $1 AND uv.expires_at > NOW()
		ORDER BY vl.sort_order DESC LIMIT 1`
	var uv domain.UserVIP
	var vl domain.VIPLevel
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&uv.ID, &uv.UserID, &uv.VIPLevelID, &uv.StartedAt, &uv.ExpiresAt,
		&uv.AutoRenew, &uv.CreatedAt,
		&vl.ID, &vl.Name, &vl.DisplayName, &vl.Price, &vl.DurationDays, &vl.BadgeURL,
		&vl.ChatColor, &vl.NameGlowColor, &vl.Privileges, &vl.SortOrder, &vl.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	uv.VIPLevel = &vl
	return &uv, nil
}

func (r *vipRepository) ListByUserID(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.UserVIP, error) {
	query := `SELECT id, user_id, vip_level_id, started_at, expires_at, auto_renew, created_at
		FROM user_vip WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.UserVIP
	for rows.Next() {
		var uv domain.UserVIP
		if err := rows.Scan(&uv.ID, &uv.UserID, &uv.VIPLevelID, &uv.StartedAt,
			&uv.ExpiresAt, &uv.AutoRenew, &uv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &uv)
	}
	return list, nil
}

func (r *vipRepository) UpdateAutoRenew(ctx context.Context, id domain.UUID, autoRenew bool) error {
	_, err := r.db.Exec(ctx, `UPDATE user_vip SET auto_renew = $1 WHERE id = $2`, autoRenew, id)
	return err
}

func (r *vipRepository) ListExpiring(ctx context.Context, before time.Time) ([]*domain.UserVIP, error) {
	query := `SELECT id, user_id, vip_level_id, started_at, expires_at, auto_renew, created_at
		FROM user_vip WHERE expires_at <= $1 AND auto_renew = true`
	rows, err := r.db.Query(ctx, query, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.UserVIP
	for rows.Next() {
		var uv domain.UserVIP
		if err := rows.Scan(&uv.ID, &uv.UserID, &uv.VIPLevelID, &uv.StartedAt,
			&uv.ExpiresAt, &uv.AutoRenew, &uv.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &uv)
	}
	return list, nil
}

func (r *vipRepository) ListEmoticonsByLevel(ctx context.Context, levelID domain.UUID) ([]*domain.VIPEmoticon, error) {
	query := `SELECT id, vip_level_id, name, code, url, created_at FROM vip_emoticons WHERE vip_level_id = $1`
	rows, err := r.db.Query(ctx, query, levelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.VIPEmoticon
	for rows.Next() {
		var e domain.VIPEmoticon
		if err := rows.Scan(&e.ID, &e.VIPLevelID, &e.Name, &e.Code, &e.URL, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &e)
	}
	return list, nil
}

func (r *vipRepository) ListAllEmoticons(ctx context.Context) ([]*domain.VIPEmoticon, error) {
	query := `SELECT id, vip_level_id, name, code, url, created_at FROM vip_emoticons ORDER BY created_at`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.VIPEmoticon
	for rows.Next() {
		var e domain.VIPEmoticon
		if err := rows.Scan(&e.ID, &e.VIPLevelID, &e.Name, &e.Code, &e.URL, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &e)
	}
	return list, nil
}

func (r *vipRepository) ListEffectsByLevel(ctx context.Context, levelID domain.UUID) ([]*domain.EntryEffect, error) {
	query := `SELECT id, vip_level_id, name, animation_url, sound_url, duration_ms, created_at
		FROM entry_effects WHERE vip_level_id = $1`
	rows, err := r.db.Query(ctx, query, levelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.EntryEffect
	for rows.Next() {
		var e domain.EntryEffect
		if err := rows.Scan(&e.ID, &e.VIPLevelID, &e.Name, &e.AnimationURL, &e.SoundURL,
			&e.DurationMs, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &e)
	}
	return list, nil
}

func (r *vipRepository) GetRandomEffect(ctx context.Context, levelID domain.UUID) (*domain.EntryEffect, error) {
	query := `SELECT id, vip_level_id, name, animation_url, sound_url, duration_ms, created_at
		FROM entry_effects WHERE vip_level_id = $1 ORDER BY RANDOM() LIMIT 1`
	var e domain.EntryEffect
	err := r.db.QueryRow(ctx, query, levelID).Scan(&e.ID, &e.VIPLevelID, &e.Name,
		&e.AnimationURL, &e.SoundURL, &e.DurationMs, &e.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

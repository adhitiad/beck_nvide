package repository

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
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

const selectStreamFields = `
	id, host_id, title, description, thumbnail_url, status, started_at, ended_at, viewer_peak, total_duration, room_id,
	room_mode, room_password_hash, entry_fee_id_r, min_level_to_enter,
	category, tags, max_resolution, is_screen_share, is_co_host_enabled,
	max_co_hosts, viewer_count, total_gift_value_id_r, like_count, share_count,
	current_pk_id, is_pk_eligible, chat_mode, chat_slow_mode_seconds,
	country_code, language, created_at, updated_at
`

func scanStream(row pgx.Row, s *domain.Stream) error {
	var (
		description      sql.NullString
		thumbnailURL     sql.NullString
		roomPasswordHash sql.NullString
		category         sql.NullString
		tags             sql.NullString
		maxResolution    sql.NullString
		chatMode         sql.NullString
		countryCode      sql.NullString
		language         sql.NullString
	)

	err := row.Scan(
		&s.ID, &s.HostID, &s.Title, &description, &thumbnailURL, &s.Status,
		&s.StartedAt, &s.EndedAt, &s.ViewerPeak, &s.TotalDuration, &s.RoomID,
		&s.RoomMode, &roomPasswordHash, &s.EntryFeeIDR, &s.MinLevelToEnter,
		&category, &tags, &maxResolution, &s.IsScreenShare, &s.IsCoHostEnabled,
		&s.MaxCoHosts, &s.ViewerCount, &s.TotalGiftValueIDR, &s.LikeCount, &s.ShareCount,
		&s.CurrentPKID, &s.IsPKEligible, &chatMode, &s.ChatSlowModeSeconds,
		&countryCode, &language, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if description.Valid {
		s.Description = description.String
	}
	if thumbnailURL.Valid {
		s.ThumbnailURL = thumbnailURL.String
	}
	if roomPasswordHash.Valid {
		s.RoomPasswordHash = roomPasswordHash.String
	}
	if category.Valid {
		s.Category = category.String
	}
	if tags.Valid {
		s.Tags = tags.String
	}
	if maxResolution.Valid {
		s.MaxResolution = maxResolution.String
	}
	if chatMode.Valid {
		s.ChatMode = chatMode.String
	}
	if countryCode.Valid {
		s.CountryCode = countryCode.String
	}
	if language.Valid {
		s.Language = language.String
	}

	return nil
}


func (r *streamRepository) Create(ctx context.Context, stream *domain.Stream) error {
	query := `
		INSERT INTO streams (
			id, host_id, title, description, thumbnail_url, status, room_id,
			room_mode, room_password_hash, entry_fee_id_r, min_level_to_enter,
			category, tags, max_resolution, is_screen_share, is_co_host_enabled,
			max_co_hosts, viewer_count, total_gift_value_id_r, like_count, share_count,
			current_pk_id, is_pk_eligible, chat_mode, chat_slow_mode_seconds,
			country_code, language, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25,
			$26, $27, NOW(), NOW()
		)
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
		stream.RoomMode,
		stream.RoomPasswordHash,
		stream.EntryFeeIDR,
		stream.MinLevelToEnter,
		stream.Category,
		stream.Tags,
		stream.MaxResolution,
		stream.IsScreenShare,
		stream.IsCoHostEnabled,
		stream.MaxCoHosts,
		stream.ViewerCount,
		stream.TotalGiftValueIDR,
		stream.LikeCount,
		stream.ShareCount,
		stream.CurrentPKID,
		stream.IsPKEligible,
		stream.ChatMode,
		stream.ChatSlowModeSeconds,
		stream.CountryCode,
		stream.Language,
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
		SET title = $1, description = $2, thumbnail_url = $3, status = $4, started_at = $5, ended_at = $6, viewer_peak = $7, total_duration = $8,
			room_mode = $9, room_password_hash = $10, entry_fee_id_r = $11, min_level_to_enter = $12,
			category = $13, tags = $14, max_resolution = $15, is_screen_share = $16, is_co_host_enabled = $17,
			max_co_hosts = $18, viewer_count = $19, total_gift_value_id_r = $20, like_count = $21, share_count = $22,
			current_pk_id = $23, is_pk_eligible = $24, chat_mode = $25, chat_slow_mode_seconds = $26,
			country_code = $27, language = $28, updated_at = NOW()
		WHERE id = $29
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
		stream.RoomMode,
		stream.RoomPasswordHash,
		stream.EntryFeeIDR,
		stream.MinLevelToEnter,
		stream.Category,
		stream.Tags,
		stream.MaxResolution,
		stream.IsScreenShare,
		stream.IsCoHostEnabled,
		stream.MaxCoHosts,
		stream.ViewerCount,
		stream.TotalGiftValueIDR,
		stream.LikeCount,
		stream.ShareCount,
		stream.CurrentPKID,
		stream.IsPKEligible,
		stream.ChatMode,
		stream.ChatSlowModeSeconds,
		stream.CountryCode,
		stream.Language,
		stream.ID,
	).Scan(&stream.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to update stream", zap.Error(err))
		return err
	}
	return nil
}

func (r *streamRepository) loadHostRelation(ctx context.Context, s *domain.Stream) {
	if s == nil {
		return
	}
	query := `
		SELECT id, username, email, avatar_url, is_verified
		FROM users
		WHERE id = $1
	`
	var (
		u domain.User
		usernameNull sql.NullString
		avatarURLNull sql.NullString
	)
	err := r.db.QueryRow(ctx, query, s.HostID).Scan(
		&u.ID, &usernameNull, &u.Email, &avatarURLNull, &u.IsVerified,
	)
	if err != nil {
		if err != pgx.ErrNoRows {
			r.logger.Warn("Failed to load host for stream", zap.Error(err), zap.String("host_id", s.HostID.String()))
		}
		return
	}
	if usernameNull.Valid {
		u.Username = usernameNull.String
	}
	if avatarURLNull.Valid {
		avatarStr := avatarURLNull.String
		u.AvatarURL = &avatarStr
	}
	s.Host = &u
}

func (r *streamRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Stream, error) {
	query := `SELECT ` + selectStreamFields + ` FROM streams WHERE id = $1`
	var s domain.Stream
	err := scanStream(r.db.QueryRow(ctx, query, id), &s)
	if err != nil {
		return nil, err
	}
	r.loadHostRelation(ctx, &s)
	return &s, nil
}

func (r *streamRepository) GetByRoomID(ctx context.Context, roomID domain.UUID) (*domain.Stream, error) {
	query := `SELECT ` + selectStreamFields + ` FROM streams WHERE room_id = $1`
	var s domain.Stream
	err := scanStream(r.db.QueryRow(ctx, query, roomID), &s)
	if err != nil {
		return nil, err
	}
	r.loadHostRelation(ctx, &s)
	return &s, nil
}

func (r *streamRepository) GetLiveByHost(ctx context.Context, hostID domain.UUID) (*domain.Stream, error) {
	query := `SELECT ` + selectStreamFields + ` FROM streams WHERE host_id = $1 AND status = 'live'`
	var s domain.Stream
	err := scanStream(r.db.QueryRow(ctx, query, hostID), &s)
	if err != nil {
		return nil, err
	}
	r.loadHostRelation(ctx, &s)
	return &s, nil
}

func (r *streamRepository) ListLive(ctx context.Context, limit, offset int) ([]*domain.Stream, error) {
	query := `
		SELECT ` + selectStreamFields + `
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

	streams := make([]*domain.Stream, 0)
	for rows.Next() {
		var s domain.Stream
		err := scanStream(rows, &s)
		if err != nil {
			return nil, err
		}
		r.loadHostRelation(ctx, &s)
		streams = append(streams, &s)
	}
	return streams, nil
}

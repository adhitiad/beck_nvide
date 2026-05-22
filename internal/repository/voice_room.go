package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type voiceRoomRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewVoiceRoomRepository(db *pgxpool.Pool, logger *zap.Logger) domain.VoiceRoomRepository {
	return &voiceRoomRepository{db: db, logger: logger}
}

func (r *voiceRoomRepository) Create(ctx context.Context, room *domain.VoiceRoom) error {
	query := `INSERT INTO voice_rooms (id, host_id, title, description, max_speakers, status,
		total_gift_value, listener_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, 0, 0, NOW()) RETURNING created_at`
	return r.db.QueryRow(ctx, query, room.ID, room.HostID, room.Title, room.Description,
		room.MaxSpeakers, room.Status).Scan(&room.CreatedAt)
}

func (r *voiceRoomRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.VoiceRoom, error) {
	query := `SELECT vr.id, vr.host_id, vr.title, vr.description, vr.max_speakers, vr.status,
		vr.total_gift_value, vr.listener_count, vr.created_at, vr.ended_at,
		u.id, u.username, u.avatar_url
		FROM voice_rooms vr JOIN users u ON vr.host_id = u.id WHERE vr.id=$1`
	var room domain.VoiceRoom
	var host domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(&room.ID, &room.HostID, &room.Title, &room.Description,
		&room.MaxSpeakers, &room.Status, &room.TotalGiftValue, &room.ListenerCount,
		&room.CreatedAt, &room.EndedAt, &host.ID, &host.Username, &host.AvatarURL)
	if err != nil {
		return nil, err
	}
	room.Host = &host
	return &room, nil
}

func (r *voiceRoomRepository) Update(ctx context.Context, room *domain.VoiceRoom) error {
	query := `UPDATE voice_rooms SET title=$1, description=$2, total_gift_value=$3, listener_count=$4 WHERE id=$5`
	_, err := r.db.Exec(ctx, query, room.Title, room.Description, room.TotalGiftValue, room.ListenerCount, room.ID)
	return err
}

func (r *voiceRoomRepository) ListActive(ctx context.Context, limit, offset int) ([]*domain.VoiceRoom, error) {
	query := `SELECT vr.id, vr.host_id, vr.title, vr.description, vr.max_speakers, vr.status,
		vr.total_gift_value, vr.listener_count, vr.created_at, vr.ended_at,
		u.id, u.username, u.avatar_url
		FROM voice_rooms vr JOIN users u ON vr.host_id = u.id
		WHERE vr.status='active' ORDER BY vr.listener_count DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.VoiceRoom
	for rows.Next() {
		var room domain.VoiceRoom
		var host domain.User
		if err := rows.Scan(&room.ID, &room.HostID, &room.Title, &room.Description,
			&room.MaxSpeakers, &room.Status, &room.TotalGiftValue, &room.ListenerCount,
			&room.CreatedAt, &room.EndedAt, &host.ID, &host.Username, &host.AvatarURL); err != nil {
			return nil, err
		}
		room.Host = &host
		list = append(list, &room)
	}
	return list, nil
}

func (r *voiceRoomRepository) EndRoom(ctx context.Context, id domain.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE voice_rooms SET status='ended', ended_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *voiceRoomRepository) AddParticipant(ctx context.Context, p *domain.VoiceRoomParticipant) error {
	query := `INSERT INTO voice_room_participants (id, room_id, user_id, role, is_muted, joined_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (room_id, user_id) DO UPDATE SET role=$4, left_at=NULL
		RETURNING joined_at`
	return r.db.QueryRow(ctx, query, p.ID, p.RoomID, p.UserID, p.Role, p.IsMuted).Scan(&p.JoinedAt)
}

func (r *voiceRoomRepository) RemoveParticipant(ctx context.Context, roomID, userID domain.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE voice_room_participants SET left_at=NOW() WHERE room_id=$1 AND user_id=$2 AND left_at IS NULL`,
		roomID, userID)
	return err
}

func (r *voiceRoomRepository) GetParticipant(ctx context.Context, roomID, userID domain.UUID) (*domain.VoiceRoomParticipant, error) {
	query := `SELECT id, room_id, user_id, role, is_muted, joined_at, left_at
		FROM voice_room_participants WHERE room_id=$1 AND user_id=$2 AND left_at IS NULL`
	var p domain.VoiceRoomParticipant
	err := r.db.QueryRow(ctx, query, roomID, userID).Scan(&p.ID, &p.RoomID, &p.UserID,
		&p.Role, &p.IsMuted, &p.JoinedAt, &p.LeftAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *voiceRoomRepository) ListParticipants(ctx context.Context, roomID domain.UUID) ([]*domain.VoiceRoomParticipant, error) {
	query := `SELECT vrp.id, vrp.room_id, vrp.user_id, vrp.role, vrp.is_muted, vrp.joined_at, vrp.left_at,
		u.id, u.username, u.avatar_url
		FROM voice_room_participants vrp JOIN users u ON vrp.user_id = u.id
		WHERE vrp.room_id=$1 AND vrp.left_at IS NULL
		ORDER BY CASE vrp.role WHEN 'host' THEN 0 WHEN 'speaker' THEN 1 ELSE 2 END`
	rows, err := r.db.Query(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.VoiceRoomParticipant
	for rows.Next() {
		var p domain.VoiceRoomParticipant
		var u domain.User
		if err := rows.Scan(&p.ID, &p.RoomID, &p.UserID, &p.Role, &p.IsMuted, &p.JoinedAt, &p.LeftAt,
			&u.ID, &u.Username, &u.AvatarURL); err != nil {
			return nil, err
		}
		p.User = &u
		list = append(list, &p)
	}
	return list, nil
}

func (r *voiceRoomRepository) UpdateParticipantRole(ctx context.Context, roomID, userID domain.UUID, role string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE voice_room_participants SET role=$1 WHERE room_id=$2 AND user_id=$3 AND left_at IS NULL`,
		role, roomID, userID)
	return err
}

func (r *voiceRoomRepository) CountSpeakers(ctx context.Context, roomID domain.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM voice_room_participants WHERE room_id=$1 AND role IN ('host','speaker') AND left_at IS NULL`,
		roomID).Scan(&count)
	return count, err
}

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type messageRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewMessageRepository creates new message repository
func NewMessageRepository(db *pgxpool.Pool, logger *zap.Logger) domain.MessageRepository {
	return &messageRepository{
		db:     db,
		logger: logger,
	}
}

func (r *messageRepository) Create(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO live_messages (id, room_id, user_id, content, type, reply_to_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query,
		message.ID, message.RoomID, message.UserID, message.Content,
		message.Type, message.ReplyToID,
	)
	if err != nil {
		r.logger.Error("Failed to create message", zap.Error(err))
		return err
	}
	return nil
}

func (r *messageRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.Message, error) {
	query := `
		SELECT id, room_id, user_id, content, type, reply_to_id, created_at, updated_at
		FROM live_messages
		WHERE id = $1
	`
	message := &domain.Message{}
	var replyToID []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&message.ID, &message.RoomID, &message.UserID, &message.Content,
		&message.Type, &replyToID, &message.CreatedAt, &message.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if replyToID != nil {
		rid, _ := domain.FromString(string(replyToID))
		message.ReplyToID = &rid
	}
	return message, nil
}

func (r *messageRepository) GetByRoomID(ctx context.Context, roomID domain.UUID, limit, offset int) ([]*domain.Message, error) {
	query := `
		SELECT id, room_id, user_id, content, type, reply_to_id, created_at, updated_at
		FROM live_messages
		WHERE room_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, roomID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0)
	for rows.Next() {
		message := &domain.Message{}
		var replyToID []byte
		err := rows.Scan(
			&message.ID, &message.RoomID, &message.UserID, &message.Content,
			&message.Type, &replyToID, &message.CreatedAt, &message.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if replyToID != nil {
			rid, _ := domain.FromString(string(replyToID))
			message.ReplyToID = &rid
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (r *messageRepository) GetRecentByRoomID(ctx context.Context, roomID domain.UUID, limit int) ([]*domain.Message, error) {
	query := `
		SELECT id, room_id, user_id, content, type, reply_to_id, created_at, updated_at
		FROM live_messages
		WHERE room_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0)
	for rows.Next() {
		message := &domain.Message{}
		var replyToID []byte
		err := rows.Scan(
			&message.ID, &message.RoomID, &message.UserID, &message.Content,
			&message.Type, &replyToID, &message.CreatedAt, &message.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if replyToID != nil {
			rid, _ := domain.FromString(string(replyToID))
			message.ReplyToID = &rid
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func (r *messageRepository) Delete(ctx context.Context, id domain.UUID) error {
	query := `DELETE FROM live_messages WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *messageRepository) CountByRoomID(ctx context.Context, roomID domain.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM live_messages WHERE room_id = $1`
	err := r.db.QueryRow(ctx, query, roomID).Scan(&count)
	return count, err
}

// Chat Room Repository
type chatRoomRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewChatRoomRepository creates new chat room repository
func NewChatRoomRepository(db *pgxpool.Pool, logger *zap.Logger) domain.ChatRoomRepository {
	return &chatRoomRepository{
		db:     db,
		logger: logger,
	}
}

func (r *chatRoomRepository) Create(ctx context.Context, room *domain.ChatRoom) error {
	query := `
		INSERT INTO live_rooms (id, name, type, target_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := r.db.Exec(ctx, query, room.ID, room.Name, room.Type, room.TargetID)
	if err != nil {
		r.logger.Error("Failed to create chat room", zap.Error(err))
		return err
	}
	return nil
}

func (r *chatRoomRepository) GetByID(ctx context.Context, id domain.UUID) (*domain.ChatRoom, error) {
	query := `
		SELECT id, name, type, target_id, created_at, updated_at
		FROM live_rooms
		WHERE id = $1
	`
	room := &domain.ChatRoom{}
	var targetID []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&room.ID, &room.Name, &room.Type, &targetID, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetID != nil {
		tid, _ := domain.FromString(string(targetID))
		room.TargetID = &tid
	}
	return room, nil
}

func (r *chatRoomRepository) GetByStreamID(ctx context.Context, streamID domain.UUID) (*domain.ChatRoom, error) {
	query := `
		SELECT id, name, type, target_id, created_at, updated_at
		FROM live_rooms
		WHERE target_id = $1 AND type = 'stream'
	`
	room := &domain.ChatRoom{}
	var targetID []byte
	err := r.db.QueryRow(ctx, query, streamID).Scan(
		&room.ID, &room.Name, &room.Type, &targetID, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetID != nil {
		tid, _ := domain.FromString(string(targetID))
		room.TargetID = &tid
	}
	return room, nil
}

func (r *chatRoomRepository) GetUserRooms(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.ChatRoom, error) {
	query := `
		SELECT cr.id, cr.name, cr.type, cr.target_id, cr.created_at, cr.updated_at
		FROM live_rooms cr
		JOIN live_room_participants rp ON cr.id = rp.room_id
		WHERE rp.user_id = $1
		ORDER BY cr.updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]*domain.ChatRoom, 0)
	for rows.Next() {
		room := &domain.ChatRoom{}
		var targetID []byte
		err := rows.Scan(
			&room.ID, &room.Name, &room.Type, &targetID, &room.CreatedAt, &room.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if targetID != nil {
			tid, _ := domain.FromString(string(targetID))
			room.TargetID = &tid
		}
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *chatRoomRepository) AddParticipant(ctx context.Context, roomID, userID domain.UUID) error {
	query := `
		INSERT INTO live_room_participants (id, room_id, user_id, joined_at)
		VALUES (uuid_generate_v7(), $1, $2, NOW())
		ON CONFLICT (room_id, user_id) DO NOTHING
	`
	_, err := r.db.Exec(ctx, query, roomID, userID)
	return err
}

func (r *chatRoomRepository) RemoveParticipant(ctx context.Context, roomID, userID domain.UUID) error {
	query := `DELETE FROM live_room_participants WHERE room_id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, roomID, userID)
	return err
}

func (r *chatRoomRepository) IsParticipant(ctx context.Context, roomID, userID domain.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM live_room_participants WHERE room_id = $1 AND user_id = $2)`
	err := r.db.QueryRow(ctx, query, roomID, userID).Scan(&exists)
	return exists, err
}

func (r *chatRoomRepository) GetPrivateRoom(ctx context.Context, user1, user2 domain.UUID) (*domain.ChatRoom, error) {
	query := `
		SELECT cr.id, cr.name, cr.type, cr.target_id, cr.created_at, cr.updated_at
		FROM live_rooms cr
		JOIN live_room_participants rp1 ON cr.id = rp1.room_id
		JOIN live_room_participants rp2 ON cr.id = rp2.room_id
		WHERE cr.type = 'private' AND rp1.user_id = $1 AND rp2.user_id = $2
		LIMIT 1
	`
	room := &domain.ChatRoom{}
	var targetID []byte
	err := r.db.QueryRow(ctx, query, user1, user2).Scan(
		&room.ID, &room.Name, &room.Type, &targetID, &room.CreatedAt, &room.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetID != nil {
		tid, _ := domain.FromString(string(targetID))
		room.TargetID = &tid
	}
	return room, nil
}

package domain

import (
	"context"
	"time"
)

// Message represents a chat message in a room
type Message struct {
	ID        UUID      `json:"id" db:"id"`
	RoomID    UUID      `json:"room_id" db:"room_id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	Type      string    `json:"type" db:"type"` // "text", "image", "system"
	ReplyToID *UUID     `json:"reply_to_id,omitempty" db:"reply_to_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Relations
	User *User `json:"user,omitempty"`
}

// ChatRoom represents a chat room (for stream chat or private chat)
type ChatRoom struct {
	ID             UUID      `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Type           string    `json:"type" db:"type"`                     // "stream", "private", "group"
	TargetID       *UUID     `json:"target_id,omitempty" db:"target_id"` // Stream ID for stream chat
	ParticipantIDs []UUID    `json:"participant_ids,omitempty"`          // For private/group chat
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// MessageRepository defines the contract for message data access
type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	GetByID(ctx context.Context, id UUID) (*Message, error)
	GetByRoomID(ctx context.Context, roomID UUID, limit, offset int) ([]*Message, error)
	GetRecentByRoomID(ctx context.Context, roomID UUID, limit int) ([]*Message, error)
	Delete(ctx context.Context, id UUID) error
	CountByRoomID(ctx context.Context, roomID UUID) (int, error)
}

// ChatRoomRepository defines the contract for chat room data access
type ChatRoomRepository interface {
	Create(ctx context.Context, room *ChatRoom) error
	GetByID(ctx context.Context, id UUID) (*ChatRoom, error)
	GetByStreamID(ctx context.Context, streamID UUID) (*ChatRoom, error)
	GetUserRooms(ctx context.Context, userID UUID, limit, offset int) ([]*ChatRoom, error)
	AddParticipant(ctx context.Context, roomID, userID UUID) error
	RemoveParticipant(ctx context.Context, roomID, userID UUID) error
	IsParticipant(ctx context.Context, roomID, userID UUID) (bool, error)
	GetPrivateRoom(ctx context.Context, user1, user2 UUID) (*ChatRoom, error)
}

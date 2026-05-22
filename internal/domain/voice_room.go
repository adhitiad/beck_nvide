package domain

import (
	"context"
	"time"
)

// Voice room statuses
const (
	VoiceRoomStatusActive = "active"
	VoiceRoomStatusEnded  = "ended"
)

// Voice room participant roles
const (
	VoiceRoleHost     = "host"
	VoiceRoleSpeaker  = "speaker"
	VoiceRoleListener = "listener"
)

// VoiceRoom represents an audio-only chat room
type VoiceRoom struct {
	ID             UUID       `json:"id"`
	HostID         UUID       `json:"host_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	MaxSpeakers    int        `json:"max_speakers"`
	Status         string     `json:"status"` // active, ended
	TotalGiftValue int64      `json:"total_gift_value"`
	ListenerCount  int        `json:"listener_count"`
	CreatedAt      time.Time  `json:"created_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`

	// Relations
	Host         *User                    `json:"host,omitempty"`
	Participants []*VoiceRoomParticipant   `json:"participants,omitempty"`
}

// VoiceRoomParticipant represents a participant in a voice room
type VoiceRoomParticipant struct {
	ID       UUID       `json:"id"`
	RoomID   UUID       `json:"room_id"`
	UserID   UUID       `json:"user_id"`
	Role     string     `json:"role"` // host, speaker, listener
	IsMuted  bool       `json:"is_muted"`
	JoinedAt time.Time  `json:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty"`

	// Relations
	User *User `json:"user,omitempty"`
}

// VoiceRoomRepository defines the contract for voice room data access
type VoiceRoomRepository interface {
	// Room
	Create(ctx context.Context, room *VoiceRoom) error
	GetByID(ctx context.Context, id UUID) (*VoiceRoom, error)
	Update(ctx context.Context, room *VoiceRoom) error
	ListActive(ctx context.Context, limit, offset int) ([]*VoiceRoom, error)
	EndRoom(ctx context.Context, id UUID) error

	// Participants
	AddParticipant(ctx context.Context, p *VoiceRoomParticipant) error
	RemoveParticipant(ctx context.Context, roomID, userID UUID) error
	GetParticipant(ctx context.Context, roomID, userID UUID) (*VoiceRoomParticipant, error)
	ListParticipants(ctx context.Context, roomID UUID) ([]*VoiceRoomParticipant, error)
	UpdateParticipantRole(ctx context.Context, roomID, userID UUID, role string) error
	CountSpeakers(ctx context.Context, roomID UUID) (int, error)
}

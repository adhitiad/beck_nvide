package domain

import (
	"context"
	"time"
)

// PaidRoom represents a private room with an entry fee
type PaidRoom struct {
	ID          UUID      `json:"id" db:"id"`
	HostID      UUID      `json:"host_id" db:"host_id"`
	Name        string    `json:"name" db:"name"`
	EntryFeeIDR int64     `json:"entry_fee_idr" db:"entry_fee_idr"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// HostDevice represents a smart device connected to a host (e.g. Lovense)
type HostDevice struct {
	ID         UUID      `json:"id" db:"id"`
	HostID     UUID      `json:"host_id" db:"host_id"`
	DeviceName string    `json:"device_name" db:"device_name"`
	DeviceID   string    `json:"device_id" db:"device_id"`
	APIToken   string    `json:"api_token" db:"api_token"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ShowRequest represents a specific custom performance request from viewer to host
type ShowRequest struct {
	ID          UUID      `json:"id" db:"id"`
	StreamID    UUID      `json:"stream_id" db:"stream_id"`
	UserID      UUID      `json:"user_id" db:"user_id"`
	Description string    `json:"description" db:"description"`
	TipsAmount  int64     `json:"tips_amount" db:"tips_amount"`
	Status      string    `json:"status" db:"status"` // "pending", "accepted", "rejected", "completed"
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// AIChatSession represents an offline companion chat session with a creator's bot
type AIChatSession struct {
	ID        UUID      `json:"id" db:"id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	HostID    UUID      `json:"host_id" db:"host_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AIChatMessage represents a message within an AI Companion chatbot session
type AIChatMessage struct {
	ID         UUID      `json:"id" db:"id"`
	SessionID  UUID      `json:"session_id" db:"session_id"`
	SenderType string    `json:"sender_type" db:"sender_type"` // "user", "ai"
	Content    string    `json:"content" db:"content"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// MonetizationRepository defines the contract for extra monetization features data access
type MonetizationRepository interface {
	// Paid Room
	CreatePaidRoom(ctx context.Context, room *PaidRoom) error
	GetPaidRoomByID(ctx context.Context, id UUID) (*PaidRoom, error)
	ListPaidRoomsByHost(ctx context.Context, hostID UUID) ([]*PaidRoom, error)

	// Interactive Toys
	SaveHostDevice(ctx context.Context, device *HostDevice) error
	GetHostDevices(ctx context.Context, hostID UUID) ([]*HostDevice, error)

	// Show Request
	CreateShowRequest(ctx context.Context, req *ShowRequest) error
	GetShowRequestByID(ctx context.Context, id UUID) (*ShowRequest, error)
	UpdateShowRequestStatus(ctx context.Context, id UUID, status string) error

	// AI Companion
	CreateAIChatSession(ctx context.Context, sess *AIChatSession) error
	GetAIChatSession(ctx context.Context, userID, hostID UUID) (*AIChatSession, error)
	SaveAIChatMessage(ctx context.Context, msg *AIChatMessage) error
	GetAIChatHistory(ctx context.Context, sessionID UUID, limit int) ([]*AIChatMessage, error)
	GetHostChatHistory(ctx context.Context, hostID UUID, limit int) ([]string, error)
}

// MonetizationUseCase defines the contract for extra monetization business logic
type MonetizationUseCase interface {
	// Paid Room
	CreatePaidRoom(ctx context.Context, hostID UUID, name string, entryFeeIDR int64) (*PaidRoom, error)
	JoinPaidRoom(ctx context.Context, userID, roomID UUID) (*PaidRoom, error)

	// Interactive Toys
	RegisterHostDevice(ctx context.Context, hostID UUID, deviceName, deviceID, apiToken string) (*HostDevice, error)
	GetHostDevices(ctx context.Context, hostID UUID) ([]*HostDevice, error)
	ControlToys(ctx context.Context, userID, streamID UUID, command string, durationSeconds int, tipsAmount int64) (string, error)

	// Show Request
	SubmitShowRequest(ctx context.Context, userID, streamID UUID, description string, tipsAmount int64) (*ShowRequest, error)
	AcceptShowRequest(ctx context.Context, hostID, requestID UUID) error
	RejectShowRequest(ctx context.Context, hostID, requestID UUID) error

	// AI Companion
	SendAIChatMessage(ctx context.Context, userID, hostID UUID, content string) (*AIChatMessage, error)
	GetAIChatHistory(ctx context.Context, userID, hostID UUID, limit int) ([]*AIChatMessage, error)
}

package domain

import (
	"context"
	"time"
)

// Call statuses
const (
	CallStatusPending  = "pending"
	CallStatusAccepted = "accepted"
	CallStatusRejected = "rejected"
	CallStatusActive   = "active"
	CallStatusEnded    = "ended"
	CallStatusFailed   = "failed"
)

// Call types
const (
	CallTypeVoice = "voice"
	CallTypeVideo = "video"
)

// HostCallRate represents call pricing for a host
type HostCallRate struct {
	ID                UUID      `json:"id" db:"id"`
	HostID            UUID      `json:"host_id" db:"host_id"`
	VoiceCallRateIDR  int64     `json:"voice_call_rate_idr" db:"voice_call_rate_idr"`
	VideoCallRateIDR  int64     `json:"video_call_rate_idr" db:"video_call_rate_idr"`
	MinDurationSeconds int       `json:"min_duration_seconds" db:"min_duration_seconds"`
	IsEnabled         bool      `json:"is_enabled" db:"is_enabled"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// PaidChatUnlock represents a one-time unlock for a conversation
type PaidChatUnlock struct {
	ID             UUID      `json:"id" db:"id"`
	ConversationID UUID      `json:"conversation_id" db:"conversation_id"`
	PayerID        UUID      `json:"payer_id" db:"payer_id"`
	RecipientID    UUID      `json:"recipient_id" db:"recipient_id"`
	AmountIDR      int64     `json:"amount_idr" db:"amount_idr"`
	Status         string    `json:"status" db:"status"` // "active", "refunded"
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// CallSession represents a 1-on-1 call between user and host
type CallSession struct {
	ID               UUID       `json:"id" db:"id"`
	HostID           UUID       `json:"host_id" db:"host_id"`
	CallerID         UUID       `json:"caller_id" db:"caller_id"`
	Type             string     `json:"type" db:"type"`
	RateIDR          int64      `json:"rate_idr" db:"rate_idr"`
	Status           string     `json:"status" db:"status"`
	StartedAt        *time.Time `json:"started_at,omitempty" db:"started_at"`
	EndedAt          *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	DurationSeconds  int        `json:"duration_seconds" db:"duration_seconds"`
	TotalChargeIDR   int64      `json:"total_charge_idr" db:"total_charge_idr"`
	PlatformFeeIDR   int64      `json:"platform_fee_idr" db:"platform_fee_idr"`
	HostEarningIDR   int64      `json:"host_earning_idr" db:"host_earning_idr"`
	EndedReason      *string    `json:"ended_reason,omitempty" db:"ended_reason"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// CallBillingTick represents a single minute charge log
type CallBillingTick struct {
	ID            UUID      `json:"id" db:"id"`
	CallSessionID UUID      `json:"call_session_id" db:"call_session_id"`
	TickNumber    int       `json:"tick_number" db:"tick_number"`
	ChargeIDR     int64     `json:"charge_idr" db:"charge_idr"`
	DeductedAt    time.Time `json:"deducted_at" db:"deducted_at"`
}

// PaidInteractionRepository defines data access for paid features
type PaidInteractionRepository interface {
	// Host Rates
	GetHostCallRate(ctx context.Context, hostID UUID) (*HostCallRate, error)
	UpsertHostCallRate(ctx context.Context, rate *HostCallRate) error

	// Paid Chat
	CreatePaidChatUnlock(ctx context.Context, unlock *PaidChatUnlock) error
	GetPaidChatUnlock(ctx context.Context, convID, payerID UUID) (*PaidChatUnlock, error)
	IsChatUnlocked(ctx context.Context, convID, payerID UUID) (bool, error)
	ListPendingRefunds(ctx context.Context, threshold time.Time) ([]*PaidChatUnlock, error)
	UpdateUnlockStatus(ctx context.Context, id UUID, status string) error

	// Paid Call
	CreateCallSession(ctx context.Context, session *CallSession) error
	GetCallSessionByID(ctx context.Context, id UUID) (*CallSession, error)
	UpdateCallSession(ctx context.Context, session *CallSession) error
	CreateBillingTick(ctx context.Context, tick *CallBillingTick) error
	GetCallHistory(ctx context.Context, userID UUID, role string, limit, offset int) ([]*CallSession, error)
}

// PaidInteractionUsecase defines business logic for paid features
type PaidInteractionUsecase interface {
	// Chat
	UnlockChat(ctx context.Context, payerID, convID UUID) error
	CheckChatUnlockStatus(ctx context.Context, payerID, convID UUID) (bool, error)
	ProcessAutoRefunds(ctx context.Context) error

	// Call
	SetHostRates(ctx context.Context, hostID UUID, voiceRate, videoRate int64, enabled bool) error
	GetHostRates(ctx context.Context, hostID UUID) (*HostCallRate, error)
	RequestCall(ctx context.Context, callerID, hostID UUID, callType string) (*CallSession, error)
	AcceptCall(ctx context.Context, sessionID UUID) error
	RejectCall(ctx context.Context, sessionID UUID, reason string) error
	EndCall(ctx context.Context, sessionID UUID, reason string) error
	ProcessBillingTick(ctx context.Context, sessionID UUID) error
}

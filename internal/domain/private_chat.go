package domain

import (
	"context"
	"encoding/json"
	"time"
)

// MessageType constants
const (
	MessageTypeText   = "text"
	MessageTypeImage  = "image"
	MessageTypeVoice  = "voice"
	MessageTypeGift   = "gift"
	MessageTypeSystem = "system"
)

// MessageStatus constants
const (
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
)

// Conversation represents a 1-on-1 chat room
type Conversation struct {
	ID            UUID       `json:"id" db:"id"`
	Type          string     `json:"type" db:"type"` // "direct", "group"
	InitiatorID   UUID       `json:"initiator_id" db:"initiator_id"`
	RecipientID   UUID       `json:"recipient_id" db:"recipient_id"`
	LastMessageID *UUID      `json:"last_message_id,omitempty" db:"last_message_id"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty" db:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`

	// Relations
	Participants []ConversationParticipant `json:"participants,omitempty"`
	LastMessage  *PrivateMessage           `json:"last_message,omitempty"`
	Recipient    *User                     `json:"recipient,omitempty"` // For convenience
}

// ConversationParticipant represents metadata for a user in a conversation
type ConversationParticipant struct {
	ID                 UUID       `json:"id" db:"id"`
	ConversationID     UUID       `json:"conversation_id" db:"conversation_id"`
	UserID             UUID       `json:"user_id" db:"user_id"`
	UnreadCount        int        `json:"unread_count" db:"unread_count"`
	IsMuted            bool       `json:"is_muted" db:"is_muted"`
	IsArchived         bool       `json:"is_archived" db:"is_archived"`
	IsPinned           bool       `json:"is_pinned" db:"is_pinned"`
	IsDeleted          bool       `json:"is_deleted" db:"is_deleted"`
	LastReadMessageID  *UUID      `json:"last_read_message_id,omitempty" db:"last_read_message_id"`
	JoinedAt           time.Time  `json:"joined_at" db:"joined_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
}

// PrivateMessage represents a message in a private conversation
type PrivateMessage struct {
	ID                UUID            `json:"id" db:"id"`
	ConversationID    UUID            `json:"conversation_id" db:"conversation_id"`
	SenderID          UUID            `json:"sender_id" db:"sender_id"`
	Type              string          `json:"type" db:"type"`
	Content           *string         `json:"content,omitempty" db:"content"`
	Metadata          json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	ReplyToMessageID  *UUID           `json:"reply_to_message_id,omitempty" db:"reply_to_message_id"`
	IsEdited          bool            `json:"is_edited" db:"is_edited"`
	IsDeleted         bool            `json:"is_deleted" db:"is_deleted"`
	IsExpired         bool            `json:"is_expired" db:"is_expired"`
	DisappearMode     string          `json:"disappear_mode" db:"disappear_mode"`
	DisappearAt       *time.Time      `json:"disappear_at,omitempty" db:"disappear_at"`
	ViewedAt          *time.Time      `json:"viewed_at,omitempty" db:"viewed_at"`
	IsScreenshot      bool            `json:"is_screenshot_detected" db:"is_screenshot_detected"`
	IsForwarded       bool            `json:"is_forwarded" db:"is_forwarded"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`

	// Relations
	Sender      *User                `json:"sender,omitempty"`
	Attachments []*MessageAttachment `json:"attachments,omitempty"`
	Reactions   []*MessageReaction   `json:"reactions,omitempty"`
	Status      []MessageStatus      `json:"status,omitempty"`
	ReplyTo     *PrivateMessage      `json:"reply_to,omitempty"`
}

// MessageReaction represents an emoji reaction to a message
type MessageReaction struct {
	ID        UUID      `json:"id" db:"id"`
	MessageID UUID      `json:"message_id" db:"message_id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// MessageView tracks who viewed a message
type MessageView struct {
	ID        UUID      `json:"id" db:"id"`
	MessageID UUID      `json:"message_id" db:"message_id"`
	ViewerID  UUID      `json:"viewer_id" db:"viewer_id"`
	ViewedAt  time.Time `json:"viewed_at" db:"viewed_at"`
}

// MessageStatus tracks delivered/read status per user
type MessageStatus struct {
	ID        UUID      `json:"id" db:"id"`
	MessageID UUID      `json:"message_id" db:"message_id"`
	UserID    UUID      `json:"user_id" db:"user_id"`
	Status    string    `json:"status" db:"status"` // "delivered", "read"
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MessageAttachment represents files attached to a message
type MessageAttachment struct {
	ID        UUID      `json:"id" db:"id"`
	MessageID UUID      `json:"message_id" db:"message_id"`
	FileName  *string   `json:"file_name,omitempty" db:"file_name"`
	FileURL   string    `json:"file_url" db:"file_url"`
	FileType  *string   `json:"file_type,omitempty" db:"file_type"`
	FileSize  *int      `json:"file_size,omitempty" db:"file_size"`
	Width     *int      `json:"width,omitempty" db:"width"`
	Height    *int      `json:"height,omitempty" db:"height"`
	Duration  *int      `json:"duration,omitempty" db:"duration"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserBlock represents a blocked user
type UserBlock struct {
	ID        UUID      `json:"id" db:"id"`
	BlockerID UUID      `json:"blocker_id" db:"blocker_id"`
	BlockedID UUID      `json:"blocked_id" db:"blocked_id"`
	Reason    *string   `json:"reason,omitempty" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// PrivateChatRepository defines the contract for private chat data access
type PrivateChatRepository interface {
	// Conversation
	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversationByID(ctx context.Context, id UUID) (*Conversation, error)
	GetConversationByParticipants(ctx context.Context, user1, user2 UUID) (*Conversation, error)
	ListConversations(ctx context.Context, userID UUID, cursorTime *time.Time, cursorID *UUID, limit int) ([]*Conversation, error)
	UpdateConversationSettings(ctx context.Context, userID, convID UUID, settings map[string]interface{}) error
	DeleteConversation(ctx context.Context, userID, convID UUID) error
	MarkAsRead(ctx context.Context, userID, convID UUID, lastMessageID UUID) error

	// Message
	CreateMessage(ctx context.Context, msg *PrivateMessage) error
	GetMessageByID(ctx context.Context, id UUID) (*PrivateMessage, error)
	ListMessages(ctx context.Context, convID UUID, cursorTime *time.Time, cursorID *UUID, limit int) ([]*PrivateMessage, error)
	UpdateMessage(ctx context.Context, msg *PrivateMessage) error
	SoftDeleteMessage(ctx context.Context, msgID UUID) error

	// Status & Attachments
	UpdateMessageStatus(ctx context.Context, msgID, userID UUID, status string) error
	BatchUpdateReadStatus(ctx context.Context, convID, userID UUID, lastMessageID UUID) error
	CreateAttachment(ctx context.Context, att *MessageAttachment) error

	// Reactions
	AddReaction(ctx context.Context, reaction *MessageReaction) error
	DeleteReaction(ctx context.Context, messageID, userID UUID, emoji string) error
	GetReactionsByMessageID(ctx context.Context, messageID UUID) ([]*MessageReaction, error)
	DeleteAllReactions(ctx context.Context, messageID, userID UUID) error

	// Disappearing Messages
	TrackView(ctx context.Context, view *MessageView) error
	UpdateMessageDisappear(ctx context.Context, messageID UUID, disappearAt *time.Time) error
	ListExpiredMessages(ctx context.Context, now time.Time) ([]*PrivateMessage, error)
	HardDeleteExpired(ctx context.Context, threshold time.Time) error

	// Blocks
	BlockUser(ctx context.Context, blockerID, blockedID UUID, reason string) error
	UnblockUser(ctx context.Context, blockerID, blockedID UUID) error
	IsBlocked(ctx context.Context, user1, user2 UUID) (bool, error)
	ListBlockedUsers(ctx context.Context, blockerID UUID) ([]*User, error)
}

// PrivateChatUsecase defines the business logic for private chat
type PrivateChatUsecase interface {
	StartConversation(ctx context.Context, initiatorID, recipientID UUID) (*Conversation, error)
	GetConversations(ctx context.Context, userID UUID, cursorTime *time.Time, cursorID *UUID, limit int) ([]*Conversation, error)
	SendMessage(ctx context.Context, senderID, convID UUID, msgType string, content string, metadata json.RawMessage, replyToID *UUID) (*PrivateMessage, error)
	GetMessages(ctx context.Context, userID, convID UUID, cursorTime *time.Time, cursorID *UUID, limit int) ([]*PrivateMessage, error)
	GetConversationByID(ctx context.Context, id UUID) (*Conversation, error)
	EditMessage(ctx context.Context, userID, msgID UUID, content string) (*PrivateMessage, error)
	DeleteMessage(ctx context.Context, userID, msgID UUID) error
	MarkConversationRead(ctx context.Context, userID, convID UUID) error
	UpdateSettings(ctx context.Context, userID, convID UUID, settings map[string]interface{}) error

	// Reactions
	ToggleReaction(ctx context.Context, userID, messageID UUID, emoji string) error
	GetReactions(ctx context.Context, messageID UUID) (map[string]interface{}, error)

	// Disappearing Messages
	MarkAsViewed(ctx context.Context, viewerID, messageID UUID) error
	NotifyScreenshot(ctx context.Context, userID, conversationID UUID) error
	ProcessExpiredMessages(ctx context.Context) error
	
	// Privacy
	BlockUser(ctx context.Context, blockerID, blockedID UUID, reason string) error
	UnblockUser(ctx context.Context, blockerID, blockedID UUID) error
	GetBlockedUsers(ctx context.Context, userID UUID) ([]*User, error)
}

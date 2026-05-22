package domain

import (
	"context"
	"time"
)

// VIP level names
const (
	VIPLevelSVIP = "svip"
	VIPLevelMVP  = "mvp"
	VIPLevelKing = "king"
)

// VIPLevel represents a VIP membership tier
type VIPLevel struct {
	ID            UUID      `json:"id"`
	Name          string    `json:"name"`
	DisplayName   string    `json:"display_name"`
	Price         int64     `json:"price"`
	DurationDays  int       `json:"duration_days"`
	BadgeURL      string    `json:"badge_url"`
	ChatColor     string    `json:"chat_color"`
	NameGlowColor string    `json:"name_glow_color"`
	Privileges    string    `json:"privileges"` // JSONB
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserVIP represents an active VIP subscription for a user
type UserVIP struct {
	ID         UUID      `json:"id"`
	UserID     UUID      `json:"user_id"`
	VIPLevelID UUID      `json:"vip_level_id"`
	StartedAt  time.Time `json:"started_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	AutoRenew  bool      `json:"auto_renew"`
	CreatedAt  time.Time `json:"created_at"`

	// Relations
	VIPLevel *VIPLevel `json:"vip_level,omitempty"`
}

// IsActive checks if the VIP subscription is still active
func (v *UserVIP) IsActive() bool {
	return time.Now().Before(v.ExpiresAt)
}

// VIPEmoticon represents an exclusive emoticon for a VIP level
type VIPEmoticon struct {
	ID         UUID      `json:"id"`
	VIPLevelID UUID      `json:"vip_level_id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
}

// EntryEffect represents a room entry animation for VIP users
type EntryEffect struct {
	ID           UUID      `json:"id"`
	VIPLevelID   UUID      `json:"vip_level_id"`
	Name         string    `json:"name"`
	AnimationURL string    `json:"animation_url"`
	SoundURL     string    `json:"sound_url"`
	DurationMs   int       `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// VIPRepository defines the contract for VIP data access
type VIPRepository interface {
	// VIP Levels
	ListLevels(ctx context.Context) ([]*VIPLevel, error)
	GetLevelByID(ctx context.Context, id UUID) (*VIPLevel, error)
	GetLevelByName(ctx context.Context, name string) (*VIPLevel, error)

	// User VIP Subscriptions
	Subscribe(ctx context.Context, uv *UserVIP) error
	GetActiveByUserID(ctx context.Context, userID UUID) (*UserVIP, error)
	ListByUserID(ctx context.Context, userID UUID, limit, offset int) ([]*UserVIP, error)
	UpdateAutoRenew(ctx context.Context, id UUID, autoRenew bool) error
	ListExpiring(ctx context.Context, before time.Time) ([]*UserVIP, error)

	// Emoticons
	ListEmoticonsByLevel(ctx context.Context, levelID UUID) ([]*VIPEmoticon, error)
	ListAllEmoticons(ctx context.Context) ([]*VIPEmoticon, error)

	// Entry Effects
	ListEffectsByLevel(ctx context.Context, levelID UUID) ([]*EntryEffect, error)
	GetRandomEffect(ctx context.Context, levelID UUID) (*EntryEffect, error)
}

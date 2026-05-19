package domain

import (
	"context"
	"time"
)

// BannedUser represents a user banned under strict policies
type BannedUser struct {
	ID                UUID      `json:"id" db:"id"`
	UserID            UUID      `json:"user_id" db:"user_id"`
	Reason            string    `json:"reason" db:"reason"` // e.g. 'lgbt_policy', 'content_violation', dll
	BannedAt          time.Time `json:"banned_at" db:"banned_at"`
	IsPermanent       bool      `json:"is_permanent" db:"is_permanent"`
	CanAppeal         bool      `json:"can_appeal" db:"can_appeal"`
	DeviceFingerprint *string   `json:"device_fingerprint,omitempty" db:"device_fingerprint"`
	IPAddress         *string   `json:"ip_address,omitempty" db:"ip_address"`
}

// BannedUserRepository defines data access methods for banned users and devices
type BannedUserRepository interface {
	BanUser(ctx context.Context, banned *BannedUser) error
	UnbanUser(ctx context.Context, userID UUID) error
	IsBanned(ctx context.Context, userID UUID) (bool, *BannedUser, error)
	IsDeviceBanned(ctx context.Context, fingerprint string) (bool, *BannedUser, error)
	IsIPBanned(ctx context.Context, ip string) (bool, *BannedUser, error)
	ListBanned(ctx context.Context, limit, offset int) ([]*BannedUser, error)
}

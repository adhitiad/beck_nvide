package domain

import (
	"context"
	"time"
)

// RefreshToken represents a refresh token stored in database
type RefreshToken struct {
	ID        UUID       `json:"id" db:"id"`
	UserID    UUID       `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"` // Never expose hash
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// IsExpired checks if token is expired
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsRevoked checks if token is revoked
func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsValid checks if token is still valid (not expired and not revoked)
func (t *RefreshToken) IsValid() bool {
	return !t.IsExpired() && !t.IsRevoked()
}

// TokenRepository defines the contract for refresh token data access
type TokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByID(ctx context.Context, id UUID) (*RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	GetActiveByUserID(ctx context.Context, userID UUID) ([]*RefreshToken, error)
	RevokeByID(ctx context.Context, id UUID) error
	RevokeAllByUserID(ctx context.Context, userID UUID) error
	DeleteExpired(ctx context.Context) error
	CountActiveByUserID(ctx context.Context, userID UUID) (int, error)
}

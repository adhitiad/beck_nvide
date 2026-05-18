package domain

import (
	"errors"
	"strings"
	"nvide-live/pkg/uuid"
)

// UUID represents a UUID as string (will be UUID v7 in production)
type UUID string

// IsZero checks if UUID is empty
func (u UUID) IsZero() bool {
	return u == ""
}

// NewUUID generates a new UUID v7
func NewUUID() UUID {
	return UUID(uuid.NewV7())
}

// NewUUIDv7 generates a new UUID v7 (timestamp-based)
func NewUUIDv7() UUID {
	return UUID(uuid.NewV7())
}

// FromString converts string to UUID
func FromString(s string) (UUID, error) {
	if s == "" {
		return "", errors.New("empty uuid string")
	}
	// Normalize: remove dashes if present
	normalized := strings.ReplaceAll(s, "-", "")
	// Validate hex string length (existing data remains backward compatible)
	if len(normalized) != 32 {
		return "", errors.New("invalid uuid format")
	}
	return UUID(normalized), nil
}

// String converts UUID to string
func (u UUID) String() string {
	return string(u)
}

// MarshalBinary implements binary marshaler (placeholder)
func (u UUID) MarshalBinary() ([]byte, error) {
	return []byte(u), nil
}

// UnmarshalBinary implements binary unmarshaler (placeholder)
func (u *UUID) UnmarshalBinary(data []byte) error {
	*u = UUID(data)
	return nil
}

// MarshalText implements text marshaler
func (u UUID) MarshalText() ([]byte, error) {
	return []byte(u), nil
}

// UnmarshalText implements text unmarshaler
func (u *UUID) UnmarshalText(text []byte) error {
	*u = UUID(text)
	return nil
}
// Metadata is a generic map for JSON data
type Metadata map[string]interface{}

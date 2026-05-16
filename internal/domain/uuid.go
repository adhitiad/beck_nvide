package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

// UUID represents a UUID as string (will be UUID v7 in production)
type UUID string

// IsZero checks if UUID is empty
func (u UUID) IsZero() bool {
	return u == ""
}

// NewUUID generates a new UUID (v4 compatible for now)
// In production, replace with UUID v7 library
func NewUUID() UUID {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	// Set version to 4
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	// Set variant to RFC 4122
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return UUID(hex.EncodeToString(bytes))
}

// NewUUIDv7 generates a new UUID v7 (timestamp-based)
func NewUUIDv7() UUID {
	// Simplified implementation for placeholder
	// In production, use a library that guarantees monotonicity
	return NewUUID()
}

// FromString converts string to UUID
func FromString(s string) (UUID, error) {
	if s == "" {
		return "", errors.New("empty uuid string")
	}
	// Normalize: remove dashes if present
	normalized := strings.ReplaceAll(s, "-", "")
	// Validate hex string length
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

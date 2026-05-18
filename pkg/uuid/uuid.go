package uuid

import (
	"github.com/google/uuid"
)

// New generates a new UUID v4 (backward compatible)
func New() string {
	return uuid.New().String()
}

// NewV7 generates a new UUID v7
func NewV7() string {
	val, err := uuid.NewV7()
	if err != nil {
		// Fallback to v4 if v7 generation fails
		return uuid.New().String()
	}
	return val.String()
}

// Parse parses a UUID string
func Parse(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// IsValid checks if a string is a valid UUID
func IsValid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

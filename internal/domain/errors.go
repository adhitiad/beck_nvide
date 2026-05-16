package domain

import "errors"

// Common domain errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrConflict           = errors.New("resource conflict")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token expired")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrValidation         = errors.New("validation error")
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// DomainError represents a domain-specific error with code
type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e DomainError) Error() string {
	return e.Code + ": " + e.Message
}

func (e DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string, err error) DomainError {
	return DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common error codes
const (
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeConflict     = "CONFLICT"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeInvalidToken = "INVALID_TOKEN"
	ErrCodeExpiredToken = "EXPIRED_TOKEN"
	ErrCodeInvalidCreds = "INVALID_CREDENTIALS"
	ErrCodeTokenRevoked = "TOKEN_REVOKED"
	ErrCodeRateLimit    = "RATE_LIMIT_EXCEEDED"
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeInternal     = "INTERNAL_ERROR"
)

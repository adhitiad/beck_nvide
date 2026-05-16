package domain

import (
	"context"
	"time"
)

// User represents the user aggregate root
type User struct {
	ID                UUID       `json:"id" db:"id"`
	Username          string     `json:"username" db:"username"`
	Email             string     `json:"email" db:"email"`
	PasswordHash      string     `json:"-" db:"password_hash"`
	RoleID            UUID       `json:"role_id" db:"role_id"`
	AvatarURL         *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	IsVerified        bool       `json:"is_verified" db:"is_verified"`
	VerificationToken *string    `json:"-" db:"verification_token"`
	ResetToken        *string    `json:"-" db:"reset_token"`
	ResetTokenExpires *time.Time `json:"-" db:"reset_token_expires_at"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`

	// Relations (not stored in users table)
	Role *Role `json:"role,omitempty"`
}

// Validate validates user fields
func (u *User) Validate() error {
	if u.Username == "" {
		return ValidationError{Field: "username", Message: "username is required"}
	}
	if len(u.Username) < 3 || len(u.Username) > 50 {
		return ValidationError{Field: "username", Message: "username must be 3-50 characters"}
	}
	if u.Email == "" {
		return ValidationError{Field: "email", Message: "email is required"}
	}
	if u.PasswordHash == "" {
		return ValidationError{Field: "password", Message: "password is required"}
	}
	if u.RoleID.IsZero() {
		return ValidationError{Field: "role_id", Message: "role is required"}
	}
	return nil
}

// IsActive checks if user account is active
func (u *User) IsActive() bool {
	return u.IsVerified
}

// HasPermission checks if user has a specific permission through their role
func (u *User) HasPermission(permissionName string, permissions []Permission) bool {
	if u.Role == nil {
		return false
	}
	return u.Role.HasPermission(permissionName, permissions)
}

// CanAccess checks if user can access a resource with specific action
func (u *User) CanAccess(resource, action string, permissions []Permission) bool {
	if u.Role == nil {
		return false
	}
	return u.Role.CanAccess(resource, action, permissions)
}

// UserRepository defines the contract for user data access
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id UUID) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	Search(ctx context.Context, query string, limit int) ([]*User, error)
	Count(ctx context.Context) (int, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdateLastLogin(ctx context.Context, id UUID) error
	FindByVerificationToken(ctx context.Context, token string) (*User, error)
	FindByResetToken(ctx context.Context, token string) (*User, error)
	ClearVerificationToken(ctx context.Context, id UUID) error
	ClearResetToken(ctx context.Context, id UUID) error
	UpdatePassword(ctx context.Context, id UUID, passwordHash string) error
}

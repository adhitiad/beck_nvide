package domain

import (
	"context"
	"time"
)

// Role represents a user role (guest, user, host, agency, admin)
type Role struct {
	ID          UUID         `json:"id" db:"id"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	Permissions []Permission `json:"permissions,omitempty"` // Many-to-many relation
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// HasPermission checks if role has a specific permission by name
func (r *Role) HasPermission(permissionName string, permissions []Permission) bool {
	for _, p := range permissions {
		if p.Name == permissionName {
			return true
		}
	}
	return false
}

// CanAccess checks if role can perform action on resource
func (r *Role) CanAccess(resource, action string, permissions []Permission) bool {
	for _, p := range permissions {
		if p.Resource == resource && p.Action == action {
			return true
		}
	}
	return false
}

// Permission represents an action that can be performed on a resource
type Permission struct {
	ID          UUID      `json:"id" db:"id"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RoleRepository defines the contract for role data access
type RoleRepository interface {
	GetByID(ctx context.Context, id UUID) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]*Role, error)
	GetPermissionsByRoleID(ctx context.Context, roleID UUID) ([]Permission, error)
	GetPermissionsByRoleName(ctx context.Context, roleName string) ([]Permission, error)
	AssignPermission(ctx context.Context, roleID, permissionID UUID) error
	RevokePermission(ctx context.Context, roleID, permissionID UUID) error
}

// PermissionRepository defines the contract for permission data access
type PermissionRepository interface {
	GetByID(ctx context.Context, id UUID) (*Permission, error)
	GetByName(ctx context.Context, name string) (*Permission, error)
	List(ctx context.Context) ([]Permission, error)
	ListByResource(ctx context.Context, resource string) ([]Permission, error)
	BulkCreate(ctx context.Context, permissions []Permission) error
}

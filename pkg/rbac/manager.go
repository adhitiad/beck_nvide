package rbac

import (
	"strings"

	"nvide-live/internal/domain"
)

// Manager handles RBAC operations
type Manager struct {
	permissions map[string][]domain.Permission // roleName -> permissions
}

// New creates a new RBAC manager
func New() *Manager {
	return &Manager{
		permissions: make(map[string][]domain.Permission),
	}
}

// LoadPermissions loads permissions for a role
func (m *Manager) LoadPermissions(roleName string, permissions []domain.Permission) {
	m.permissions[roleName] = permissions
}

// HasPermission checks if role has specific permission
func (m *Manager) HasPermission(roleName, permissionName string) bool {
	if perms, ok := m.permissions[roleName]; ok {
		for _, p := range perms {
			if p.Name == permissionName {
				return true
			}
		}
	}
	return false
}

// CanAccess checks if role can perform action on resource
func (m *Manager) CanAccess(roleName, resource, action string) bool {
	if perms, ok := m.permissions[roleName]; ok {
		for _, p := range perms {
			if p.Resource == resource && p.Action == action {
				return true
			}
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func (m *Manager) GetPermissions(roleName string) []domain.Permission {
	return m.permissions[roleName]
}

// HasResourcePermission checks if user can access resource with any action
func (m *Manager) HasResourcePermission(roleName, resource string) bool {
	if perms, ok := m.permissions[roleName]; ok {
		for _, p := range perms {
			if p.Resource == resource {
				return true
			}
		}
	}
	return false
}

// ParseRequiredPermission parses permission string "resource:action"
func ParseRequiredPermission(perm string) (resource, action string) {
	parts := strings.Split(perm, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return perm, ""
}

package middleware

import (
	"net/http"

	"go.uber.org/zap"
	"nvide-live/pkg/rbac"
)

// RBACMiddleware handles role-based access control
type RBACMiddleware struct {
	rbacManager *rbac.Manager
	logger      *zap.Logger
}

// NewRBACMiddleware creates new RBAC middleware
func NewRBACMiddleware(rbacManager *rbac.Manager, logger *zap.Logger) *RBACMiddleware {
	return &RBACMiddleware{
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// RequirePermission checks if user has required permission
func (m *RBACMiddleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get role from context
			role, ok := GetRoleFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Role not found in context")
				return
			}

			// Check permission
			if !m.rbacManager.HasPermission(role, permission) {
				writeJSONError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireResourceAccess checks if user can access resource with action
func (m *RBACMiddleware) RequireResourceAccess(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRoleFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Role not found in context")
				return
			}

			if !m.rbacManager.CanAccess(role, resource, action) {
				writeJSONError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole restricts access to specific roles
func (m *RBACMiddleware) RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := GetRoleFromContext(r.Context())
			if !ok {
				writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Role not found in context")
				return
			}

			for _, allowedRole := range allowedRoles {
				if role == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeJSONError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient role privileges")
		})
	}
}

// Optional RBAC that doesn't block if fails, just adds context
func (m *RBACMiddleware) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just pass through, auth middleware already validated token
		next.ServeHTTP(w, r)
	})
}

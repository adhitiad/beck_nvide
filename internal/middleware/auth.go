package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
	"nvide-live/pkg/auth"
	"nvide-live/pkg/logger"
	"nvide-live/pkg/redis"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService *auth.Service
	redisClient *redis.Client
	logger      *zap.Logger
}

// NewAuthMiddleware creates new auth middleware
func NewAuthMiddleware(authService *auth.Service, redisClient *redis.Client, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// Middleware validates JWT token and injects user info into context
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
			return
		}

		// Extract token (Bearer <token>)
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
			return
		}
		tokenString := parts[1]

		// Validate token
		claims, err := m.authService.ValidateToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				writeJSONError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
			} else {
				writeJSONError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token")
			}
			return
		}

		// Check blacklist in Redis
		tokenHash := hashToken(tokenString)
		blacklistKey := "blacklist:token:" + tokenHash
		ctx := r.Context()
		exists, err := m.redisClient.Exists(ctx, blacklistKey)
		if err != nil {
			m.logger.Error("Failed to check token blacklist", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			return
		}
		if exists > 0 {
			writeJSONError(w, http.StatusUnauthorized, "TOKEN_REVOKED", "Token has been revoked")
			return
		}

		// Parse user ID
		userID, err := domain.FromString(claims.UserID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user ID in token")
			return
		}

		// Inject user info into context
		ctx = contextWithValue(ctx, userIDKey, claims.UserID)
		ctx = contextWithValue(ctx, usernameKey, claims.Username)
		ctx = contextWithValue(ctx, emailKey, claims.Email)
		ctx = contextWithValue(ctx, roleKey, claims.Role)
		ctx = contextWithValue(ctx, roleIDKey, claims.RoleID)
		ctx = contextWithValue(ctx, userIDUUIDKey, userID)

		// Inject claims map & logger correlation (Problem 6)
		userClaimsMap := map[string]interface{}{
			"user_id":   claims.UserID,
			"username":  claims.Username,
			"email":     claims.Email,
			"role":      claims.Role,
			"role_id":   claims.RoleID,
		}
		ctx = context.WithValue(ctx, "user_claims", userClaimsMap)
		ctx = context.WithValue(ctx, UserClaimsKey, userClaimsMap)
		ctx = context.WithValue(ctx, logger.UserIDKey, claims.UserID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper: hash token for comparison with blacklist
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// context helpers

type contextKey string

const (
	UserClaimsKey contextKey = "user_claims"
	userIDKey     contextKey = "user_id"
	usernameKey   contextKey = "username"
	emailKey      contextKey = "email"
	roleKey       contextKey = "role"
	roleIDKey     contextKey = "role_id"
	userIDUUIDKey contextKey = "user_id_uuid"
)

func contextWithValue(ctx context.Context, key contextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

// writeJSONError writes error response as JSON
func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

// GetUserIDFromContext retrieves user ID from context
func GetUserIDFromContext(ctx context.Context) (domain.UUID, bool) {
	val, ok := ctx.Value(userIDUUIDKey).(domain.UUID)
	return val, ok
}

// GetRoleFromContext retrieves user role from context
func GetRoleFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(roleKey).(string)
	return val, ok
}

// GetEmailFromContext retrieves email from context
func GetEmailFromContext(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(emailKey).(string)
	return val, ok
}

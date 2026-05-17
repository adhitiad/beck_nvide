package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"nvide-live/internal/domain"
)

type AuthHandler struct {
	db          *sql.DB
	redisClient *redis.Client
}

func NewAuthHandler(db *sql.DB, redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{
		db:          db,
		redisClient: redisClient,
	}
}

// BetterAuthMiddleware validates sessions from Better Auth with Redis caching
func (h *AuthHandler) BetterAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Get session token from cookie
			cookie, err := r.Cookie("better-auth.session_token")
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			sessionToken := cookie.Value
			sessionKey := "auth:session:" + sessionToken

			var userIDStr string
			
			// 2. Try to get from Redis Cache first
			cachedUID, err := h.redisClient.Get(r.Context(), sessionKey).Result()
			if err == nil {
				userIDStr = cachedUID
			} else {
				// 3. Cache miss: Query Database
				var expiresAt time.Time
				query := `SELECT "user_id", "expires_at" FROM "sessions" WHERE "token" = $1 LIMIT 1`
				err = h.db.QueryRowContext(r.Context(), query, sessionToken).Scan(&userIDStr, &expiresAt)
				
				if err != nil || time.Now().After(expiresAt) {
					next.ServeHTTP(w, r)
					return
				}

				// 4. Save to Redis for 10 minutes to optimize future requests
				h.redisClient.Set(r.Context(), sessionKey, userIDStr, 10*time.Minute)
			}

			// 5. Convert to Domain UUID and inject into Context
			userID, err := domain.FromString(userIDStr)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), userIDUUIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"nvide-live/internal/domain"
)

// BetterAuthMiddleware validates sessions from Better Auth
func BetterAuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get session token from cookie
			cookie, err := r.Cookie("better-auth.session_token")
			if err != nil {
				// No session cookie, allow through (protected routes will handle)
				next.ServeHTTP(w, r)
				return
			}

			sessionToken := cookie.Value
			if sessionToken == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Query Better Auth session table
			var userIDStr string
			var expiresAt time.Time
			
			// Note: GORM maps model Session to table 'sessions'
			query := `SELECT "UserID", "ExpiresAt" FROM "sessions" WHERE "Token" = $1 LIMIT 1`
			err = db.QueryRow(query, sessionToken).Scan(&userIDStr, &expiresAt)
			
			if err != nil {
				if err != sql.ErrNoRows {
					// Database error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				// Session not found
				next.ServeHTTP(w, r)
				return
			}

			// Check if session is expired
			if time.Now().After(expiresAt) {
				next.ServeHTTP(w, r)
				return
			}

			// Convert string ID to domain.UUID
			userID, err := domain.FromString(userIDStr)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Add userID to context
			ctx := context.WithValue(r.Context(), userIDUUIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

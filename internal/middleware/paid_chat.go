package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

// PaidChatMiddleware handles pay-to-chat enforcement
type PaidChatMiddleware struct {
	repo   domain.PaidInteractionRepository
	logger *zap.Logger
}

// NewPaidChatMiddleware creates new paid chat middleware
func NewPaidChatMiddleware(repo domain.PaidInteractionRepository, logger *zap.Logger) *PaidChatMiddleware {
	return &PaidChatMiddleware{
		repo:   repo,
		logger: logger,
	}
}

// Middleware checks if a conversation is unlocked for the current user
func (m *PaidChatMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get conversation ID from route vars
		vars := mux.Vars(r)
		convIDStr := vars["id"]
		if convIDStr == "" {
			next.ServeHTTP(w, r)
			return
		}

		convID, err := domain.FromString(convIDStr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Get user ID from context
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
			return
		}

		// Role-based bypass: Host can send messages for free
		role, _ := GetRoleFromContext(r.Context())
		if role == "host" || role == "admin" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if conversation is unlocked
		unlocked, err := m.repo.IsChatUnlocked(r.Context(), convID, userID)
		if err != nil {
			m.logger.Error("Failed to check chat unlock status", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			return
		}

		if !unlocked {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":    "PAYMENT_REQUIRED",
				"message":  "You must unlock this conversation to send messages",
				"price":    3500,
				"currency": "IDR",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"nvide-live/internal/usecase"
)

type ClipQuotaMiddleware struct {
	subUseCase *usecase.ClipSubscriptionUseCase
	logger     *zap.Logger
}

func NewClipQuotaMiddleware(subUseCase *usecase.ClipSubscriptionUseCase, logger *zap.Logger) *ClipQuotaMiddleware {
	return &ClipQuotaMiddleware{
		subUseCase: subUseCase,
		logger:     logger,
	}
}

func (m *ClipQuotaMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only intercept POST /streams/{id}/clips or POST /clips/generate
		isClipGen := r.Method == http.MethodPost && (strings.Contains(r.URL.Path, "/clips") || strings.Contains(r.URL.Path, "/clip"))
		if !isClipGen {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()
		userID, ok := GetUserIDFromContext(ctx)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User context not found")
			return
		}

		// Check subscription status
		status, err := m.subUseCase.GetStatus(ctx, userID)
		if err != nil {
			m.logger.Error("Failed to fetch subscription status in middleware", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error checking subscription")
			return
		}

		isSubscribed, _ := status["is_subscribed"].(bool)
		if !isSubscribed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired) // 402 Payment Required
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "NO_ACTIVE_SUBSCRIPTION",
				"message": "You must have an active VIP AI Clip subscription to generate clips.",
				"details": status,
			})
			return
		}

		quotaLeft, _ := status["quota_left"].(int)
		if quotaLeft <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden) // 403 Forbidden
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "QUOTA_EXCEEDED",
				"message": "Your VIP AI Clip subscription quota has been fully consumed for this billing cycle.",
				"details": status,
			})
			return
		}

		// Proceed to handler
		next.ServeHTTP(w, r)
	})
}

package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
)

type ClipSubscriptionHandler struct {
	subUseCase *usecase.ClipSubscriptionUseCase
	logger     *zap.Logger
}

func NewClipSubscriptionHandler(
	subUseCase *usecase.ClipSubscriptionUseCase,
	logger *zap.Logger,
) *ClipSubscriptionHandler {
	return &ClipSubscriptionHandler{
		subUseCase: subUseCase,
		logger:     logger,
	}
}

func (h *ClipSubscriptionHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *ClipSubscriptionHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

// ListPlans lists all subscription packages, applying "Promo Host Pertama" if applicable
func (h *ClipSubscriptionHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	plans, err := h.subUseCase.ListPlans(r.Context(), userID)
	if err != nil {
		h.logger.Error("ListPlans handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve subscription packages")
		return
	}

	h.writeJSON(w, http.StatusOK, plans)
}

// Subscribe processes choosing package and debiting wallet
func (h *ClipSubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var req struct {
		PlanID string `json:"plan_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlanID == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "plan_id is required")
		return
	}

	planID, err := domain.FromString(req.PlanID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PLAN_ID", "Invalid plan ID format")
		return
	}

	sub, err := h.subUseCase.Subscribe(r.Context(), userID, planID)
	if err != nil {
		h.logger.Error("Subscribe handler failed", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "SUBSCRIBE_FAILED", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "VIP AI Clip Subscription successfully activated!",
		"subscription": sub,
	})
}

// GetStatus returns the current VIP subscription sisa hari dan kuota
func (h *ClipSubscriptionHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	status, err := h.subUseCase.GetStatus(r.Context(), userID)
	if err != nil {
		h.logger.Error("GetStatus handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check subscription status")
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

// GetHistory returns subscription purchase history
func (h *ClipSubscriptionHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	limit := 10
	offset := 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	history, err := h.subUseCase.GetHistory(r.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error("GetHistory handler failed", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve subscription history")
		return
	}

	h.writeJSON(w, http.StatusOK, history)
}

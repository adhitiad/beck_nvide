package delivery

import (
	"encoding/json"
	"net/http"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// LikeContent handles content liking
func (h *Handler) LikeContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentID  string `json:"content_id"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// Parse content ID
	contentID, err := domain.FromString(req.ContentID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONTENT_ID", "Invalid content ID")
		return
	}

	if err := h.likeUseCase.LikeContent(r.Context(), userID, contentID, req.ContentType); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Content liked"})
}

// UnlikeContent handles content unliking
func (h *Handler) UnlikeContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentID  string `json:"content_id"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// Parse content ID
	contentID, err := domain.FromString(req.ContentID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONTENT_ID", "Invalid content ID")
		return
	}

	if err := h.likeUseCase.UnlikeContent(r.Context(), userID, contentID, req.ContentType); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Content unliked"})
}

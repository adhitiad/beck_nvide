package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// RecommendationHandler menangani request HTTP untuk sistem rekomendasi personalisasi AI
type RecommendationHandler struct {
	useCase domain.RecommendationUseCaseInterface
	logger  *zap.Logger
}

// NewRecommendationHandler membuat instance baru dari RecommendationHandler
func NewRecommendationHandler(useCase domain.RecommendationUseCaseInterface, logger *zap.Logger) *RecommendationHandler {
	return &RecommendationHandler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *RecommendationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *RecommendationHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// TrackInteraction handles POST /recommendations/interactions
func (h *RecommendationHandler) TrackInteraction(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	var req struct {
		StreamID        *string                `json:"stream_id"`
		InteractionType string                 `json:"interaction_type"` // 'watch', 'like', 'comment', 'gift'
		DurationSeconds int                    `json:"duration_seconds"`
		Metadata        map[string]interface{} `json:"metadata"`
		IsIncognito     bool                   `json:"is_incognito"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload tidak valid")
		return
	}

	if req.InteractionType == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_TYPE", "Tipe interaksi diperlukan")
		return
	}

	var streamID *domain.UUID
	if req.StreamID != nil && *req.StreamID != "" {
		parsed, err := domain.FromString(*req.StreamID)
		if err == nil {
			streamID = &parsed
		}
	}

	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	if req.IsIncognito {
		req.Metadata["is_incognito"] = true
	}

	err := h.useCase.TrackInteraction(r.Context(), userID, streamID, req.InteractionType, req.DurationSeconds, req.Metadata)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal menyimpan interaksi")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "success",
		"message": "Interaction processed successfully",
	})
}

// GetRecommendedStreams handles GET /recommendations/streams
func (h *RecommendationHandler) GetRecommendedStreams(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	limit := 10
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	streams, err := h.useCase.GetRecommendedStreams(r.Context(), userID, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mendapatkan rekomendasi stream")
		return
	}

	h.writeJSON(w, http.StatusOK, streams)
}

// GetRecommendedVODs handles GET /recommendations/vods
func (h *RecommendationHandler) GetRecommendedVODs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User tidak terautentikasi")
		return
	}

	limit := 10
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	vods, err := h.useCase.GetRecommendedVODs(r.Context(), userID, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mendapatkan rekomendasi VOD")
		return
	}

	h.writeJSON(w, http.StatusOK, vods)
}

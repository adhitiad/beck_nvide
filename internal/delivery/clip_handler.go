package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

// ClipHandler menangani request HTTP untuk klip AI otomatis
type ClipHandler struct {
	useCase domain.ClipUseCaseInterface
	logger  *zap.Logger
}

// NewClipHandler membuat instance baru dari ClipHandler
func NewClipHandler(useCase domain.ClipUseCaseInterface, logger *zap.Logger) *ClipHandler {
	return &ClipHandler{
		useCase: useCase,
		logger:  logger,
	}
}

func (h *ClipHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *ClipHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// TriggerClip handles POST /api/v1/streams/{stream_id}/clips/trigger
func (h *ClipHandler) TriggerClip(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamIDStr := vars["stream_id"]
	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "ID stream tidak valid")
		return
	}

	var req struct {
		Title      string `json:"title"`
		StartOffset int    `json:"start_offset_seconds"`
		Duration   int    `json:"duration_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Format payload salah")
		return
	}

	if req.Duration <= 0 {
		req.Duration = 30 // durasi default 30 detik
	}
	if req.Title == "" {
		req.Title = "Manual Highlight Clip"
	}

	// Trigger manual klip dengan skor baseline tinggi (misal 100.0)
	clip, err := h.useCase.GenerateClip(r.Context(), streamID, req.StartOffset, req.Duration, req.Title, 100.0)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal memotong klip stream")
		return
	}

	h.writeJSON(w, http.StatusOK, clip)
}

// GetStreamClips handles GET /api/v1/streams/{stream_id}/clips
func (h *ClipHandler) GetStreamClips(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamIDStr := vars["stream_id"]
	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "ID stream tidak valid")
		return
	}

	clips, err := h.useCase.GetStreamClips(r.Context(), streamID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mendapatkan klip stream")
		return
	}

	h.writeJSON(w, http.StatusOK, clips)
}

// GetTrendingClips handles GET /api/v1/clips/trending
func (h *ClipHandler) GetTrendingClips(w http.ResponseWriter, r *http.Request) {
	limit := 10
	offset := 0

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	clips, err := h.useCase.GetTrendingClips(r.Context(), limit, offset)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mendapatkan klip viral")
		return
	}

	h.writeJSON(w, http.StatusOK, clips)
}

// Hook ke chat websocket/like/gift
// Handler ini bisa dipanggil oleh WebSocket chat hub atau monetization service secara internal
func (h *ClipHandler) TrackInteractionInternal(ctx context.Context, streamID domain.UUID, eventType string, weight float64) {
	_ = h.useCase.RegisterInteractionEvent(ctx, streamID, eventType, weight)
}

package delivery

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

type VODHandler struct {
	vodUseCase domain.VODUseCaseInterface
	logger     *zap.Logger
}

func NewVODHandler(vodUseCase domain.VODUseCaseInterface, logger *zap.Logger) *VODHandler {
	return &VODHandler{
		vodUseCase: vodUseCase,
		logger:     logger,
	}
}

func (h *VODHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *VODHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error_code": code,
		"message":    message,
	})
}

// UploadVOD handles multipart upload
func (h *VODHandler) UploadVOD(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// 500 MB max
	r.ParseMultipartForm(500 << 20)

	file, header, err := r.FormFile("video")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_FILE", "Video file is required")
		return
	}
	defer file.Close()

	// Validate file size (max 500MB)
	if header.Size > 500*1024*1024 {
		h.writeError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "Video size exceeds 500MB limit")
		return
	}

	// Validate file format (.mp4 only)
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".mp4" {
		h.writeError(w, http.StatusBadRequest, "INVALID_FORMAT", "Only MP4 format is allowed")
		return
	}

	title := r.FormValue("title")
	if title == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_TITLE", "Title is required")
		return
	}
	description := r.FormValue("description")
	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = domain.VODVisibilityPublic
	}

	// Save to temp
	tempFile, err := os.CreateTemp("", "upload-*.mp4")
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create temp file")
		return
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save temp file")
		return
	}
	tempFilePath := tempFile.Name()

	vod, err := h.vodUseCase.UploadVideo(r.Context(), userID, title, description, visibility, tempFilePath, header.Filename)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusAccepted, vod)
}

func (h *VODHandler) GetVODList(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var vods []*domain.VODMedia
	var err error

	if userIDStr != "" {
		userID, err := domain.FromString(userIDStr)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid User ID")
			return
		}
		vods, err = h.vodUseCase.ListUserVODs(r.Context(), userID, limit, offset)
	} else {
		// List all public VODs if no user_id provided
		vods, err = h.vodUseCase.ListPublicVODs(r.Context(), limit, offset)
	}

	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, vods)
}

func (h *VODHandler) GetVODDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vodID, err := domain.FromString(vars["vod_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "Invalid VOD ID")
		return
	}

	vod, err := h.vodUseCase.GetVODDetail(r.Context(), vodID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "VOD not found")
		return
	}

	h.writeJSON(w, http.StatusOK, vod)
}

type UpdateVisibilityRequest struct {
	Visibility string `json:"visibility"`
}

func (h *VODHandler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	vodID, err := domain.FromString(vars["vod_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "Invalid VOD ID")
		return
	}

	var req UpdateVisibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.vodUseCase.UpdateVisibility(r.Context(), vodID, userID, req.Visibility); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Visibility updated"})
}

func (h *VODHandler) DeleteVOD(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	vars := mux.Vars(r)
	vodID, err := domain.FromString(vars["vod_id"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_VOD_ID", "Invalid VOD ID")
		return
	}

	if err := h.vodUseCase.DeleteVOD(r.Context(), vodID, userID); err != nil {
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "VOD deleted"})
}

// ServeStorage serves static files from local storage (for testing)
func (h *VODHandler) ServeStorage(w http.ResponseWriter, r *http.Request) {
	// Not safe for production, simple static file server
	vars := mux.Vars(r)
	filepathStr := vars["filepath"]
	http.ServeFile(w, r, filepath.Join("./uploads", filepathStr))
}

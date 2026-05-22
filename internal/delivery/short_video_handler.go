package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
)

type ShortVideoHandler struct {
	videoUC *usecase.ShortVideoUseCase
}

func NewShortVideoHandler(videoUC *usecase.ShortVideoUseCase) *ShortVideoHandler {
	return &ShortVideoHandler{videoUC: videoUC}
}

func (h *ShortVideoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		VideoURL     string   `json:"video_url"`
		ThumbnailURL string   `json:"thumbnail_url"`
		Caption      string   `json:"caption"`
		Duration     int      `json:"duration"`
		Tags         []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input := domain.CreateShortVideoInput{
		VideoURL:     req.VideoURL,
		ThumbnailURL: req.ThumbnailURL,
		Caption:      req.Caption,
		Duration:     req.Duration,
		Tags:         req.Tags,
	}
	video, err := h.videoUC.Upload(r.Context(), userID, input)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, video)
}

func (h *ShortVideoHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	limit, offset := getPagination(r)
	videos, err := h.videoUC.GetFeed(r.Context(), userID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, videos)
}

func (h *ShortVideoHandler) GetTrending(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPagination(r)
	videos, err := h.videoUC.GetTrending(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, videos)
}

func (h *ShortVideoHandler) GetVideo(w http.ResponseWriter, r *http.Request) {
	videoID := domain.UUID(mux.Vars(r)["id"])
	_ = h.videoUC.RecordView(r.Context(), videoID)
	video, err := h.videoUC.GetByID(r.Context(), videoID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, video)
}

func (h *ShortVideoHandler) Like(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	videoID := domain.UUID(mux.Vars(r)["id"])
	if err := h.videoUC.Like(r.Context(), userID, videoID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "liked"})
}

func (h *ShortVideoHandler) Unlike(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	videoID := domain.UUID(mux.Vars(r)["id"])
	if err := h.videoUC.Unlike(r.Context(), userID, videoID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "unliked"})
}

func (h *ShortVideoHandler) Comment(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	videoID := domain.UUID(mux.Vars(r)["id"])
	var req struct {
		Content  string  `json:"content"`
		ParentID *string `json:"parent_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var parentID *domain.UUID
	if req.ParentID != nil {
		pid := domain.UUID(*req.ParentID)
		parentID = &pid
	}
	comment, err := h.videoUC.Comment(r.Context(), userID, videoID, req.Content, parentID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, comment)
}

func (h *ShortVideoHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	videoID := domain.UUID(mux.Vars(r)["id"])
	limit, offset := getPagination(r)
	comments, err := h.videoUC.GetComments(r.Context(), videoID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, comments)
}

func (h *ShortVideoHandler) Share(w http.ResponseWriter, r *http.Request) {
	videoID := domain.UUID(mux.Vars(r)["id"])
	_ = h.videoUC.Share(r.Context(), videoID)
	respondJSON(w, http.StatusOK, map[string]string{"status": "shared"})
}

func (h *ShortVideoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	videoID := domain.UUID(mux.Vars(r)["id"])
	if err := h.videoUC.DeleteVideo(r.Context(), userID, videoID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ShortVideoHandler) GetUserVideos(w http.ResponseWriter, r *http.Request) {
	uid := domain.UUID(mux.Vars(r)["userId"])
	limit, offset := getPagination(r)
	videos, err := h.videoUC.GetUserVideos(r.Context(), uid, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, videos)
}

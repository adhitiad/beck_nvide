package delivery

import (
	"net/http"
	"strconv"
	"encoding/json"

	"github.com/gorilla/mux"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
)

// CreateStory handles story creation
func (h *Handler) CreateStory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content   string `json:"content"`
		MediaType string `json:"media_type"`
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

	story, err := h.storyUseCase.CreateStory(r.Context(), userID, &usecase.CreateStoryRequest{
		Content:   req.Content,
		MediaType: req.MediaType,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toStoryDTO(story))
}

// GetStory gets a story by ID
func (h *Handler) GetStory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Story ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STORY_ID", "Invalid story ID")
		return
	}

	story, err := h.storyUseCase.GetStory(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toStoryDTO(story))
}

// GetUserStories gets stories by user ID
func (h *Handler) GetUserStories(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// Parse query parameters
	limit := 10
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	stories, err := h.storyUseCase.GetUserStories(r.Context(), userID, limit, offset)
	if err != nil {
		h.handleError(w, err)
		return
	}

	storyDTOs := make([]*StoryDTO, len(stories))
	for i, story := range stories {
		storyDTOs[i] = toStoryDTO(story)
	}
	h.writeJSON(w, http.StatusOK, storyDTOs)
}

// ViewStory records a story view
func (h *Handler) ViewStory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Story ID is required")
		return
	}

	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STORY_ID", "Invalid story ID")
		return
	}

	if err := h.storyUseCase.ViewStory(r.Context(), userID, id); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Story viewed"})
}

// GetFeedStories gets stories from followed users
func (h *Handler) GetFeedStories(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// Parse query parameters
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	stories, err := h.storyUseCase.GetFeedStories(r.Context(), userID, limit)
	if err != nil {
		h.handleError(w, err)
		return
	}

	storyDTOs := make([]*StoryDTO, len(stories))
	for i, story := range stories {
		storyDTOs[i] = toStoryDTO(story)
	}
	h.writeJSON(w, http.StatusOK, storyDTOs)
}

// DeleteStory deletes a story
func (h *Handler) DeleteStory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Story ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STORY_ID", "Invalid story ID")
		return
	}

	if err := h.storyUseCase.DeleteStory(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Story deleted"})
}

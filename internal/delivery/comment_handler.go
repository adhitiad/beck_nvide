package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
)

// CreateComment handles comment creation
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentID  string `json:"content_id"`
		ContentType string `json:"content_type"`
		ParentID   *string `json:"parent_id,omitempty"`
		Content    string `json:"content"`
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

	// Parse parent ID if provided
	var parentID *domain.UUID
	if req.ParentID != nil {
		pid, err := domain.FromString(*req.ParentID)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_PARENT_ID", "Invalid parent ID")
			return
		}
		parentID = &pid
	}

	comment, err := h.commentUseCase.CreateComment(r.Context(), userID, &usecase.CreateCommentRequest{
		ContentID:   contentID,
		ContentType: req.ContentType,
		ParentID:    parentID,
		Content:     req.Content,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toCommentDTO(comment))
}

// GetComments gets comments for content
func (h *Handler) GetComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	contentIDStr := vars["content_id"]
	contentType := vars["content_type"]
	if contentIDStr == "" || contentType == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Content ID and type are required")
		return
	}

	contentID, err := domain.FromString(contentIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONTENT_ID", "Invalid content ID")
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

	comments, err := h.commentUseCase.GetComments(r.Context(), contentID, contentType, limit, offset)
	if err != nil {
		h.handleError(w, err)
		return
	}

	commentDTOs := make([]*CommentDTO, len(comments))
	for i, comment := range comments {
		commentDTOs[i] = toCommentDTO(comment)
	}
	h.writeJSON(w, http.StatusOK, commentDTOs)
}

// GetComment gets a comment by ID
func (h *Handler) GetComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Comment ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_COMMENT_ID", "Invalid comment ID")
		return
	}

	comment, err := h.commentUseCase.GetComment(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toCommentDTO(comment))
}

// LikeComment handles comment liking
func (h *Handler) LikeComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Comment ID is required")
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
		h.writeError(w, http.StatusBadRequest, "INVALID_COMMENT_ID", "Invalid comment ID")
		return
	}

	if err := h.commentUseCase.LikeComment(r.Context(), userID, id); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Comment liked"})
}

// UnlikeComment handles comment unliking
func (h *Handler) UnlikeComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Comment ID is required")
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
		h.writeError(w, http.StatusBadRequest, "INVALID_COMMENT_ID", "Invalid comment ID")
		return
	}

	if err := h.commentUseCase.UnlikeComment(r.Context(), userID, id); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Comment unliked"})
}

// UpdateComment updates a comment
func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Comment ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_COMMENT_ID", "Invalid comment ID")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.commentUseCase.UpdateComment(r.Context(), id, req.Content); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Comment updated"})
}

// DeleteComment deletes a comment
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Comment ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_COMMENT_ID", "Invalid comment ID")
		return
	}

	if err := h.commentUseCase.DeleteComment(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Comment deleted"})
}

// GetReplies gets replies to a comment
func (h *Handler) GetReplies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parentIDStr := vars["id"]
	if parentIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Parent ID is required")
		return
	}

	parentID, err := domain.FromString(parentIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_PARENT_ID", "Invalid parent ID")
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

	comments, err := h.commentUseCase.GetReplies(r.Context(), parentID, limit, offset)
	if err != nil {
		h.handleError(w, err)
		return
	}

	commentDTOs := make([]*CommentDTO, len(comments))
	for i, comment := range comments {
		commentDTOs[i] = toCommentDTO(comment)
	}
	h.writeJSON(w, http.StatusOK, commentDTOs)
}

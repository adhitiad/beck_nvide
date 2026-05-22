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

// SendMessage handles sending a message
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomID    string `json:"room_id"`
		Content   string `json:"content"`
		Type      string `json:"type"`
		ReplyToID *string `json:"reply_to_id,omitempty"`
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

	// Parse room ID
	roomID, err := domain.FromString(req.RoomID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "Invalid room ID")
		return
	}

	// Parse reply to ID if provided
	var replyToID *domain.UUID
	if req.ReplyToID != nil {
		rid, err := domain.FromString(*req.ReplyToID)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "INVALID_REPLY_TO_ID", "Invalid reply to ID")
			return
		}
		replyToID = &rid
	}

	message, err := h.messageUseCase.SendMessage(r.Context(), userID, &usecase.SendMessageRequest{
		RoomID:    roomID,
		Content:   req.Content,
		Type:      req.Type,
		ReplyToID: replyToID,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toMessageDTO(message))
}

// GetMessages gets messages from a room
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Room ID is required")
		return
	}

	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "Invalid room ID")
		return
	}

	// Parse query parameters
	limit := 20
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

	messages, err := h.messageUseCase.GetMessages(r.Context(), roomID, limit, offset)
	if err != nil {
		h.handleError(w, err)
		return
	}

	messageDTOs := make([]*MessageDTO, len(messages))
	for i, message := range messages {
		messageDTOs[i] = toMessageDTO(message)
	}
	h.writeJSON(w, http.StatusOK, messageDTOs)
}

// GetRecentMessages gets recent messages from a room
func (h *Handler) GetRecentMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Room ID is required")
		return
	}

	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "Invalid room ID")
		return
	}

	// Parse query parameter
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	messages, err := h.messageUseCase.GetRecentMessages(r.Context(), roomID, limit)
	if err != nil {
		h.handleError(w, err)
		return
	}

	messageDTOs := make([]*MessageDTO, len(messages))
	for i, message := range messages {
		messageDTOs[i] = toMessageDTO(message)
	}
	h.writeJSON(w, http.StatusOK, messageDTOs)
}

// JoinRoom handles joining a chat room
func (h *Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Room ID is required")
		return
	}

	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "Invalid room ID")
		return
	}

	if err := h.messageUseCase.JoinRoom(r.Context(), roomID, userID); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Joined room"})
}

// LeaveRoom handles leaving a chat room
func (h *Handler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Room ID is required")
		return
	}

	// Get user ID from context
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	roomID, err := domain.FromString(roomIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_ROOM_ID", "Invalid room ID")
		return
	}

	if err := h.messageUseCase.LeaveRoom(r.Context(), roomID, userID); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Left room"})
}

// GetMessage gets a message by ID
func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Message ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_MESSAGE_ID", "Invalid message ID")
		return
	}

	message, err := h.messageUseCase.GetMessage(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toMessageDTO(message))
}

// DeleteMessage deletes a message
func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Message ID is required")
		return
	}

	id, err := domain.FromString(idStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_MESSAGE_ID", "Invalid message ID")
		return
	}

	if err := h.messageUseCase.DeleteMessage(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Message deleted"})
}

// GetOrCreateRoom gets or creates a chat room for stream
func (h *Handler) GetOrCreateRoom(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	streamIDStr := vars["stream_id"]
	if streamIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Stream ID is required")
		return
	}

	streamID, err := domain.FromString(streamIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_STREAM_ID", "Invalid stream ID")
		return
	}

	room, err := h.messageUseCase.GetOrCreateRoom(r.Context(), streamID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, room)
}

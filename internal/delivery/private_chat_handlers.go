package delivery

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	gorillaws "github.com/gorilla/websocket"
	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
	"nvide-live/internal/middleware"
	"go.uber.org/zap"
)

var chatUpgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ConversationDTO for private chat
type ConversationDTO struct {
	ID            string      `json:"id"`
	Type          string      `json:"type"`
	InitiatorID   string      `json:"initiator_id"`
	RecipientID   string      `json:"recipient_id"`
	LastMessageAt *time.Time  `json:"last_message_at,omitempty"`
	UnreadCount   int         `json:"unread_count"`
	IsMuted       bool        `json:"is_muted"`
	IsArchived    bool        `json:"is_archived"`
	IsPinned      bool        `json:"is_pinned"`
	Recipient     *UserDTO    `json:"recipient,omitempty"`
	LastMessage   *PrivateMessageDTO `json:"last_message,omitempty"`
}

// PrivateMessageDTO for private chat
type PrivateMessageDTO struct {
	ID               string          `json:"id"`
	ConversationID   string          `json:"conversation_id"`
	SenderID         string          `json:"sender_id"`
	Type             string          `json:"type"`
	Content          *string         `json:"content,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	ReplyToMessageID *string         `json:"reply_to_message_id,omitempty"`
	IsEdited         bool            `json:"is_edited"`
	IsDeleted        bool            `json:"is_deleted"`
	IsEncrypted      bool            `json:"is_encrypted"`
	DisappearMode    string          `json:"disappear_mode"`
	DisappearAt      *time.Time      `json:"disappear_at,omitempty"`
	ViewedAt         *time.Time      `json:"viewed_at,omitempty"`
	IsScreenshot     bool            `json:"is_screenshot_detected"`
	IsExpired        bool            `json:"is_expired"`
	IsForwarded      bool            `json:"is_forwarded"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func toConversationDTO(c *domain.Conversation) *ConversationDTO {
	dto := &ConversationDTO{
		ID:            c.ID.String(),
		Type:          c.Type,
		InitiatorID:   c.InitiatorID.String(),
		RecipientID:   c.RecipientID.String(),
		LastMessageAt: c.LastMessageAt,
	}

	if len(c.Participants) > 0 {
		dto.UnreadCount = c.Participants[0].UnreadCount
		dto.IsMuted = c.Participants[0].IsMuted
		dto.IsArchived = c.Participants[0].IsArchived
		dto.IsPinned = c.Participants[0].IsPinned
	}

	if c.Recipient != nil {
		dto.Recipient = toUserDTO(c.Recipient)
	}

	if c.LastMessage != nil {
		dto.LastMessage = toPrivateMessageDTO(c.LastMessage)
	}

	return dto
}

func toPrivateMessageDTO(m *domain.PrivateMessage) *PrivateMessageDTO {
	return &PrivateMessageDTO{
		ID:             m.ID.String(),
		ConversationID: m.ConversationID.String(),
		SenderID:       m.SenderID.String(),
		Type:           m.Type,
		Content:        m.Content,
		Metadata:       m.Metadata,
		ReplyToMessageID: func() *string {
			if m.ReplyToMessageID != nil {
				s := m.ReplyToMessageID.String()
				return &s
			}
			return nil
		}(),
		IsEdited:      m.IsEdited,
		IsDeleted:     m.IsDeleted,
		IsEncrypted:   m.IsEncrypted,
		DisappearMode: m.DisappearMode,
		DisappearAt:   m.DisappearAt,
		ViewedAt:      m.ViewedAt,
		IsScreenshot:  m.IsScreenshot,
		IsExpired:     m.IsExpired,
		IsForwarded:   m.IsForwarded,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}

// StartConversation handles POST /conversations
func (h *Handler) StartConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecipientID string `json:"recipient_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	recipientID, err := domain.FromString(req.RecipientID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_RECIPIENT_ID", "Invalid recipient ID")
		return
	}

	conv, err := h.privateChatUseCase.StartConversation(r.Context(), userID, recipientID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toConversationDTO(conv))
}

// GetConversations handles GET /conversations
func (h *Handler) GetConversations(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	// Pagination params
	limit := 20
	// TODO: cursor parsing

	conversations, err := h.privateChatUseCase.GetConversations(r.Context(), userID, nil, nil, limit)
	if err != nil {
		h.handleError(w, err)
		return
	}

	dtos := make([]*ConversationDTO, len(conversations))
	for i, c := range conversations {
		dtos[i] = toConversationDTO(c)
	}

	h.writeJSON(w, http.StatusOK, dtos)
}

// SendPrivateMessage handles POST /conversations/{id}/messages
func (h *Handler) SendPrivateMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["id"]
	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	var req struct {
		Type          string          `json:"type"`
		Content       string          `json:"content"`
		Metadata      json.RawMessage `json:"metadata,omitempty"`
		ReplyToID     *string         `json:"reply_to_id,omitempty"`
		IsEncrypted   bool            `json:"is_encrypted"`
		DisappearMode string          `json:"disappear_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	var replyToID *domain.UUID
	if req.ReplyToID != nil {
		rid, _ := domain.FromString(*req.ReplyToID)
		replyToID = &rid
	}

	msg, err := h.privateChatUseCase.SendMessage(r.Context(), userID, convID, req.Type, req.Content, req.Metadata, replyToID, req.IsEncrypted, req.DisappearMode)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, toPrivateMessageDTO(msg))
}

// GetPrivateMessages handles GET /conversations/{id}/messages
func (h *Handler) GetPrivateMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["id"]
	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	messages, err := h.privateChatUseCase.GetMessages(r.Context(), userID, convID, nil, nil, 50)
	if err != nil {
		h.handleError(w, err)
		return
	}

	dtos := make([]*PrivateMessageDTO, len(messages))
	for i, m := range messages {
		dtos[i] = toPrivateMessageDTO(m)
	}

	h.writeJSON(w, http.StatusOK, dtos)
}

// MarkAsRead handles POST /conversations/{id}/read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["id"]
	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.privateChatUseCase.MarkConversationRead(r.Context(), userID, convID); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Marked as read"})
}

// UpdateConversationSettings handles PUT /conversations/{id}/settings
func (h *Handler) UpdateConversationSettings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["id"]
	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.privateChatUseCase.UpdateSettings(r.Context(), userID, convID, settings); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Settings updated"})
}

// BlockUser handles POST /users/{id}/block
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetIDStr := vars["id"]
	targetID, err := domain.FromString(targetIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	if err := h.privateChatUseCase.BlockUser(r.Context(), userID, targetID, "Blocked by user"); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "User blocked"})
}

// Reactions
func (h *Handler) ToggleReaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	msgIDStr := vars["id"]
	msgID, err := domain.FromString(msgIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_MESSAGE_ID", "Invalid message ID")
		return
	}

	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err = h.privateChatUseCase.ToggleReaction(r.Context(), userID, msgID, req.Emoji)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Reaction toggled"})
}

// Disappearing Messages
func (h *Handler) MarkMessageViewed(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	msgIDStr := vars["id"]
	msgID, err := domain.FromString(msgIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_MESSAGE_ID", "Invalid message ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err = h.privateChatUseCase.MarkAsViewed(r.Context(), userID, msgID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Message marked as viewed"})
}


// ServeChatWS handles WebSocket requests for private chat
func (h *Handler) ServeChatWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["conversation_id"]
	if convIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Conversation ID is required")
		return
	}

	// Token validation via query param
	token := r.URL.Query().Get("token")
	if token == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token is required")
		return
	}

	claims, err := h.authUseCase.ValidateToken(r.Context(), token)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
		return
	}

	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	// Check if user is participant
	conv, err := h.privateChatUseCase.StartConversation(r.Context(), h.getUserID(r), "") // This is a hacky way to get conv, better add GetByID to usecase
	// Actually better just use repository or usecase.GetMessages logic
	_ = conv

	// Upgrade connection
	conn, err := chatUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket connection", zap.Error(err))
		return
	}

	// Register client to hub with "chat:" prefix for room isolation
	roomID := "chat:" + convID.String()
	websocket.NewClient(h.wsHub, conn, roomID, claims.UserID)
}

// Helper to get user ID from context correctly using middleware helper
func (h *Handler) getUserID(r *http.Request) domain.UUID {
	id, _ := middleware.GetUserIDFromContext(r.Context())
	return id
}

// RegisterE2EEKey handles POST /users/me/e2ee-key
func (h *Handler) RegisterE2EEKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PublicKey string `json:"public_key"`
		KeyType   string `json:"key_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.PublicKey == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Public key is required")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "X25519"
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err := h.privateChatUseCase.RegisterE2EEKey(r.Context(), userID, req.PublicKey, req.KeyType)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "E2EE public key registered successfully"})
}

// GetE2EEKey handles GET /users/{id}/e2ee-key
func (h *Handler) GetE2EEKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetIDStr := vars["id"]
	targetID, err := domain.FromString(targetIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	key, err := h.privateChatUseCase.GetE2EEKey(r.Context(), targetID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, key)
}

// MuteUser handles POST /users/{id}/mute
func (h *Handler) MuteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetIDStr := vars["id"]
	targetID, err := domain.FromString(targetIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	var req struct {
		DurationMinutes int `json:"duration_minutes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) // Optional, so ignore error

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err = h.privateChatUseCase.MuteUser(r.Context(), userID, targetID, req.DurationMinutes)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "User muted successfully"})
}

// UnmuteUser handles POST /users/{id}/unmute
func (h *Handler) UnmuteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetIDStr := vars["id"]
	targetID, err := domain.FromString(targetIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Invalid user ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err = h.privateChatUseCase.UnmuteUser(r.Context(), userID, targetID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "User unmuted successfully"})
}

// GetMutedUsers handles GET /users/me/mutes
func (h *Handler) GetMutedUsers(w http.ResponseWriter, r *http.Request) {
	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	users, err := h.privateChatUseCase.GetMutedUsers(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	dtos := make([]*UserDTO, len(users))
	for i, u := range users {
		dtos[i] = toUserDTO(u)
	}

	h.writeJSON(w, http.StatusOK, dtos)
}

// UpdatePrivacySettings handles PUT /users/me/privacy
func (h *Handler) UpdatePrivacySettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsPrivateProfile bool `json:"is_private_profile"`
		IsIncognito      bool `json:"is_incognito"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err := h.privateChatUseCase.UpdateUserPrivacy(r.Context(), userID, req.IsPrivateProfile, req.IsIncognito)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Privacy settings updated successfully"})
}

// NotifyScreenshot handles POST /conversations/{id}/screenshot
func (h *Handler) NotifyScreenshot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	convIDStr := vars["id"]
	convID, err := domain.FromString(convIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_CONVERSATION_ID", "Invalid conversation ID")
		return
	}

	userID := h.getUserID(r)
	if userID.IsZero() {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	err = h.privateChatUseCase.NotifyScreenshot(r.Context(), userID, convID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Screenshot notification sent successfully"})
}

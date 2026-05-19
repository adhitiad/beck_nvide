package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
	"nvide-live/internal/websocket"
	gorillaws "github.com/gorilla/websocket"
)

// Handler wraps HTTP handlers with dependencies
type Handler struct {
	authUseCase    *usecase.AuthUseCase
	userUseCase    *usecase.UserUseCase
	storyUseCase   *usecase.StoryUseCase
	commentUseCase *usecase.CommentUseCase
	likeUseCase    *usecase.LikeUseCase
	messageUseCase *usecase.MessageUseCase
	privateChatUseCase domain.PrivateChatUsecase
	paidInteractionUseCase domain.PaidInteractionUsecase
	bookingUseCase domain.BookingUsecase
	offerUseCase   domain.OfferUsecase
	locationUseCase domain.LocationUsecase
	liveScheduleUseCase domain.LiveScheduleUseCase
	waitRoomHub    *websocket.WaitRoomHub
	wsHub          *websocket.Hub
	logger         *zap.Logger
}

// NewHandler creates new HTTP handlers
func NewHandler(
	authUseCase *usecase.AuthUseCase,
	userUseCase *usecase.UserUseCase,
	storyUseCase *usecase.StoryUseCase,
	commentUseCase *usecase.CommentUseCase,
	likeUseCase *usecase.LikeUseCase,
	messageUseCase *usecase.MessageUseCase,
	privateChatUseCase domain.PrivateChatUsecase,
	paidInteractionUseCase domain.PaidInteractionUsecase,
	bookingUseCase domain.BookingUsecase,
	offerUseCase domain.OfferUsecase,
	locationUseCase domain.LocationUsecase,
	liveScheduleUseCase domain.LiveScheduleUseCase,
	waitRoomHub *websocket.WaitRoomHub,
	wsHub *websocket.Hub,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		authUseCase:   authUseCase,
		userUseCase:   userUseCase,
		storyUseCase:  storyUseCase,
		commentUseCase: commentUseCase,
		likeUseCase:   likeUseCase,
		messageUseCase: messageUseCase,
		privateChatUseCase: privateChatUseCase,
		paidInteractionUseCase: paidInteractionUseCase,
		bookingUseCase:         bookingUseCase,
		offerUseCase:           offerUseCase,
		locationUseCase:        locationUseCase,
		liveScheduleUseCase:    liveScheduleUseCase,
		waitRoomHub:            waitRoomHub,
		wsHub:                  wsHub,
		logger:                 logger,
	}
}

// RegisterRequest untuk registrasi
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest untuk login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse untuk response auth
type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         *UserDTO    `json:"user"`
}

// UserDTO untuk response user
type UserDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	IsVerified bool `json:"is_verified"`
}

// StoryDTO untuk response story
type StoryDTO struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	MediaType string    `json:"media_type"`
	ExpiresAt time.Time `json:"expires_at"`
	ViewCount int       `json:"view_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommentDTO untuk response comment
type CommentDTO struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ContentID  string    `json:"content_id"`
	ContentType string   `json:"content_type"`
	ParentID   *string   `json:"parent_id,omitempty"`
	Content    string    `json:"content"`
	LikeCount  int       `json:"like_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MessageDTO untuk response message
type MessageDTO struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	ReplyToID *string   `json:"reply_to_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toUserDTO converts domain.User to UserDTO
func toUserDTO(user *domain.User) *UserDTO {
	if user == nil {
		return nil
	}
	
	roleName := ""
	if user.Role != nil {
		roleName = user.Role.Name
	}

	return &UserDTO{
		ID:         user.ID.String(),
		Username:   user.Username,
		Email:      user.Email,
		Role:       roleName,
		AvatarURL:  user.AvatarURL,
		IsVerified: user.IsVerified,
	}
}

// toStoryDTO converts domain.Story to StoryDTO
func toStoryDTO(story *domain.Story) *StoryDTO {
	return &StoryDTO{
		ID:        story.ID.String(),
		UserID:    story.UserID.String(),
		Content:   story.Content,
		MediaType: story.MediaType,
		ExpiresAt: story.ExpiresAt,
		ViewCount: story.ViewCount,
		CreatedAt: story.CreatedAt,
		UpdatedAt: story.UpdatedAt,
	}
}

// toCommentDTO converts domain.Comment to CommentDTO
func toCommentDTO(comment *domain.Comment) *CommentDTO {
	return &CommentDTO{
		ID:         comment.ID.String(),
		UserID:     comment.UserID.String(),
		ContentID:  comment.ContentID.String(),
		ContentType: comment.ContentType,
		ParentID:   func() *string {
			if comment.ParentID != nil {
				s := comment.ParentID.String()
				return &s
			}
			return nil
		}(),
		Content:    comment.Content,
		LikeCount:  comment.LikeCount,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
	}
}

// toMessageDTO converts domain.Message to MessageDTO
func toMessageDTO(message *domain.Message) *MessageDTO {
	return &MessageDTO{
		ID:        message.ID.String(),
		RoomID:    message.RoomID.String(),
		UserID:    message.UserID.String(),
		Content:   message.Content,
		Type:      message.Type,
		ReplyToID: func() *string {
			if message.ReplyToID != nil {
				s := message.ReplyToID.String()
				return &s
			}
			return nil
		}(),
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
	}
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Basic validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Username, email, and password are required")
		return
	}

	user, err := h.authUseCase.Register(r.Context(), &usecase.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &AuthResponse{
		User: toUserDTO(user),
	}

	h.writeJSON(w, http.StatusCreated, response)
}

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

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		return true
	},
}

// ServeWS handles WebSocket requests
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomIDStr := vars["room_id"]
	if roomIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Room ID is required")
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

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket connection", zap.Error(err))
		return
	}

	// Check if user is participant of the room
	roomID, err := domain.FromString(roomIDStr)
	if err == nil {
		userID, err := domain.FromString(claims.UserID)
		if err == nil {
			// Actually we should check via messageUseCase if they can join,
			// but for now we just register them to the hub.
			// The usecase will be used if needed.
			
			// Auto join room if not already a participant (for streams usually)
			_ = h.messageUseCase.JoinRoom(r.Context(), roomID, userID)
		}
	}

	// Register client to hub
	websocket.NewClient(h.wsHub, conn, roomIDStr, claims.UserID)
}

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	accessToken, refreshToken, user, err := h.authUseCase.Login(r.Context(), &usecase.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserDTO(user),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Refresh handles token refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required")
		return
	}

	newAccessToken, newRefreshToken, user, err := h.authUseCase.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         toUserDTO(user),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get tokens from headers or body
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	// Try to decode body if present
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if err := h.authUseCase.Logout(r.Context(), accessToken, req.RefreshToken); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Me returns current user profile
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	user, err := h.userUseCase.GetProfile(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toUserDTO(user))
}

// writeJSON writes JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes error response in standard format
func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

// writeErrorLocalized writes a localized error response
func (h *Handler) writeErrorLocalized(ctx context.Context, w http.ResponseWriter, status int, code, translationKey string, defaultMessage string) {
	msg := middleware.T(ctx, translationKey)
	if msg == translationKey { // translation key not found, fallback to defaultMessage
		msg = defaultMessage
	}
	h.writeError(w, status, code, msg)
}

// handleError handles domain errors and converts to HTTP response
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	h.handleErrorCtx(context.Background(), w, err)
}

// handleErrorCtx handles domain errors and converts to localized HTTP response
func (h *Handler) handleErrorCtx(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case err == domain.ErrNotFound:
		h.writeErrorLocalized(ctx, w, http.StatusNotFound, domain.ErrCodeNotFound, "not_found", "Resource not found")
	case err == domain.ErrConflict:
		h.writeErrorLocalized(ctx, w, http.StatusConflict, domain.ErrCodeConflict, "conflict", err.Error())
	case err == domain.ErrUnauthorized:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeUnauthorized, "unauthorized", err.Error())
	case err == domain.ErrForbidden:
		h.writeErrorLocalized(ctx, w, http.StatusForbidden, domain.ErrCodeForbidden, "forbidden", err.Error())
	case err == domain.ErrInvalidToken:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeInvalidToken, "invalid_token", err.Error())
	case err == domain.ErrExpiredToken:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeExpiredToken, "expired_token", err.Error())
	case err == domain.ErrTokenRevoked:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeTokenRevoked, "token_revoked", err.Error())
	case err == domain.ErrRateLimitExceeded:
		h.writeErrorLocalized(ctx, w, http.StatusTooManyRequests, domain.ErrCodeRateLimit, "rate_limit", err.Error())
	case err == domain.ErrInvalidCredentials:
		h.writeErrorLocalized(ctx, w, http.StatusUnauthorized, domain.ErrCodeInvalidCreds, "invalid_credentials", err.Error())
	case err != nil:
		if _, ok := err.(domain.ValidationError); ok {
			h.writeErrorLocalized(ctx, w, http.StatusBadRequest, domain.ErrCodeValidation, "validation_error", err.Error())
		} else {
			h.logger.Error("Unhandled error", zap.Error(err))
			h.writeErrorLocalized(ctx, w, http.StatusInternalServerError, domain.ErrCodeInternal, "internal_error", "Internal server error")
		}
	}
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
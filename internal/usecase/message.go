package usecase

import (
	"context"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
)

// MessageUseCase handles message business logic
type MessageUseCase struct {
	messageRepo  domain.MessageRepository
	chatRoomRepo domain.ChatRoomRepository
	userRepo     domain.UserRepository
	logger       *zap.Logger
}

// NewMessageUseCase creates new message usecase
func NewMessageUseCase(
	messageRepo domain.MessageRepository,
	chatRoomRepo domain.ChatRoomRepository,
	userRepo domain.UserRepository,
	logger *zap.Logger,
) *MessageUseCase {
	return &MessageUseCase{
		messageRepo:  messageRepo,
		chatRoomRepo: chatRoomRepo,
		userRepo:     userRepo,
		logger:       logger,
	}
}

// SendMessageRequest represents message sending request
type SendMessageRequest struct {
	RoomID    domain.UUID  `json:"room_id"`
	Content   string       `json:"content"`
	Type      string       `json:"type"` // "text", "image", "system"
	ReplyToID *domain.UUID `json:"reply_to_id,omitempty"`
}

// SendMessage sends a message to a room
func (uc *MessageUseCase) SendMessage(ctx context.Context, userID domain.UUID, req *SendMessageRequest) (*domain.Message, error) {
	// Check if user is participant
	isParticipant, err := uc.chatRoomRepo.IsParticipant(ctx, req.RoomID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, domain.NewDomainError(domain.ErrCodeForbidden, "not a participant", nil)
	}

	message := &domain.Message{
		ID:        domain.NewUUID(),
		RoomID:    req.RoomID,
		UserID:    userID,
		Content:   req.Content,
		Type:      req.Type,
		ReplyToID: req.ReplyToID,
	}

	if err := uc.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	uc.logger.Info("Message sent", zap.String("message_id", message.ID.String()), zap.String("room_id", req.RoomID.String()))
	return message, nil
}

// GetMessages gets messages from a room
func (uc *MessageUseCase) GetMessages(ctx context.Context, roomID domain.UUID, limit, offset int) ([]*domain.Message, error) {
	return uc.messageRepo.GetByRoomID(ctx, roomID, limit, offset)
}

// GetRecentMessages gets recent messages from a room
func (uc *MessageUseCase) GetRecentMessages(ctx context.Context, roomID domain.UUID, limit int) ([]*domain.Message, error) {
	return uc.messageRepo.GetRecentByRoomID(ctx, roomID, limit)
}

// GetMessage gets a message by ID
func (uc *MessageUseCase) GetMessage(ctx context.Context, id domain.UUID) (*domain.Message, error) {
	return uc.messageRepo.GetByID(ctx, id)
}

// DeleteMessage deletes a message
func (uc *MessageUseCase) DeleteMessage(ctx context.Context, id domain.UUID) error {
	return uc.messageRepo.Delete(ctx, id)
}

// GetOrCreateRoom gets or creates a chat room for stream
func (uc *MessageUseCase) GetOrCreateRoom(ctx context.Context, streamID domain.UUID) (*domain.ChatRoom, error) {
	room, err := uc.chatRoomRepo.GetByStreamID(ctx, streamID)
	if err == nil {
		return room, nil
	}

	// Create new room
	room = &domain.ChatRoom{
		ID:       domain.NewUUID(),
		Name:     "Stream Chat",
		Type:     "stream",
		TargetID: &streamID,
	}

	if err := uc.chatRoomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

// GetOrCreatePrivateRoom gets or creates a private chat room between user and host
func (uc *MessageUseCase) GetOrCreatePrivateRoom(ctx context.Context, userID, hostID domain.UUID) (*domain.ChatRoom, error) {
	// Check if already exists
	room, err := uc.chatRoomRepo.GetPrivateRoom(ctx, userID, hostID)
	if err == nil {
		return room, nil
	}

	// Create new room
	room = &domain.ChatRoom{
		ID:   domain.NewUUID(),
		Name: "Private Chat",
		Type: "private",
	}

	if err := uc.chatRoomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// Add both as participants
	_ = uc.chatRoomRepo.AddParticipant(ctx, room.ID, userID)
	_ = uc.chatRoomRepo.AddParticipant(ctx, room.ID, hostID)

	return room, nil
}

// JoinRoom adds user to a chat room
func (uc *MessageUseCase) JoinRoom(ctx context.Context, roomID, userID domain.UUID) error {
	return uc.chatRoomRepo.AddParticipant(ctx, roomID, userID)
}

// LeaveRoom removes user from a chat room
func (uc *MessageUseCase) LeaveRoom(ctx context.Context, roomID, userID domain.UUID) error {
	return uc.chatRoomRepo.RemoveParticipant(ctx, roomID, userID)
}

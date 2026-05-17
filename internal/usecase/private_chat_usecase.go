package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

var emojiWhitelist = map[string]bool{
	"👍": true, "❤️": true, "😂": true, "😮": true, "😢": true, "😡": true,
	"🎉": true, "🔥": true, "👏": true, "💯": true, "🙏": true, "😍": true,
	"🤔": true, "😭": true, "🤣": true, "🥰": true, "😊": true, "🥳": true,
	"👀": true, "🫡": true, "🤝": true, "✨": true, "🙌": true, "💪": true,
	"👻": true, "🎁": true, "💔": true, "🤡": true, "👋": true, "🌟": true,
}

type privateChatUsecase struct {
	repo       domain.PrivateChatRepository
	userRepo   domain.UserRepository
	redis      *redis.Client
	logger     *zap.Logger
}

func NewPrivateChatUsecase(
	repo domain.PrivateChatRepository,
	userRepo domain.UserRepository,
	redis *redis.Client,
	logger *zap.Logger,
) domain.PrivateChatUsecase {
	return &privateChatUsecase{
		repo:     repo,
		userRepo: userRepo,
		redis:    redis,
		logger:   logger,
	}
}

func (u *privateChatUsecase) StartConversation(ctx context.Context, initiatorID, recipientID domain.UUID) (*domain.Conversation, error) {
	// Check if already exists
	existing, err := u.repo.GetConversationByParticipants(ctx, initiatorID, recipientID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Check if blocked
	blocked, err := u.repo.IsBlocked(ctx, initiatorID, recipientID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, errors.New("cannot message this user: you are blocked or have blocked them")
	}

	// Create new
	conv := &domain.Conversation{
		ID:          domain.NewUUIDv7(),
		Type:        "direct",
		InitiatorID: initiatorID,
		RecipientID: recipientID,
	}

	err = u.repo.CreateConversation(ctx, conv)
	if err != nil {
		return nil, err
	}

	return conv, nil
}

func (u *privateChatUsecase) GetConversations(ctx context.Context, userID domain.UUID, cursorTime *time.Time, cursorID *domain.UUID, limit int) ([]*domain.Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	return u.repo.ListConversations(ctx, userID, cursorTime, cursorID, limit)
}

func (u *privateChatUsecase) SendMessage(ctx context.Context, senderID, convID domain.UUID, msgType string, content string, metadata json.RawMessage, replyToID *domain.UUID) (*domain.PrivateMessage, error) {
	// Validate conversation membership
	conv, err := u.repo.GetConversationByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv.InitiatorID != senderID && conv.RecipientID != senderID {
		return nil, errors.New("not a participant of this conversation")
	}

	// Check block list
	otherUserID := conv.InitiatorID
	if senderID == conv.InitiatorID {
		otherUserID = conv.RecipientID
	}
	
	blocked, err := u.repo.IsBlocked(ctx, senderID, otherUserID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, errors.New("cannot send message: blocking active")
	}

	// Create message
	msg := &domain.PrivateMessage{
		ID:               domain.NewUUIDv7(),
		ConversationID:   convID,
		SenderID:         senderID,
		Type:             msgType,
		Content:          &content,
		Metadata:         metadata,
		ReplyToMessageID: replyToID,
	}

	err = u.repo.CreateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Update Redis unread count
	unreadKey := fmt.Sprintf("chat:unread:%s:%s", convID, otherUserID)
	_ = u.redis.GetClient().Incr(ctx, unreadKey)

	// Set online status for sender (heartbeat)
	onlineKey := fmt.Sprintf("chat:online:%s", senderID)
	_ = u.redis.Set(ctx, onlineKey, "online", 5*time.Minute)

	// TODO: Broadcast via WebSocket
	// TODO: Trigger push notification if recipient offline

	return msg, nil
}

func (u *privateChatUsecase) GetMessages(ctx context.Context, userID, convID domain.UUID, cursorTime *time.Time, cursorID *domain.UUID, limit int) ([]*domain.PrivateMessage, error) {
	// Validate membership
	conv, err := u.repo.GetConversationByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv.InitiatorID != userID && conv.RecipientID != userID {
		return nil, errors.New("not a participant")
	}

	if limit <= 0 {
		limit = 50
	}

	return u.repo.ListMessages(ctx, convID, cursorTime, cursorID, limit)
}

func (u *privateChatUsecase) GetConversationByID(ctx context.Context, id domain.UUID) (*domain.Conversation, error) {
	return u.repo.GetConversationByID(ctx, id)
}

func (u *privateChatUsecase) EditMessage(ctx context.Context, userID, msgID domain.UUID, content string) (*domain.PrivateMessage, error) {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return nil, err
	}

	if msg.SenderID != userID {
		return nil, errors.New("cannot edit someone else's message")
	}

	// Check edit window (5 minutes)
	if time.Since(msg.CreatedAt) > 5*time.Minute {
		return nil, errors.New("edit window (5 minutes) has expired")
	}

	msg.Content = &content
	msg.IsEdited = true
	
	err = u.repo.UpdateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (u *privateChatUsecase) DeleteMessage(ctx context.Context, userID, msgID domain.UUID) error {
	msg, err := u.repo.GetMessageByID(ctx, msgID)
	if err != nil {
		return err
	}

	if msg.SenderID != userID {
		return errors.New("cannot delete someone else's message")
	}

	return u.repo.SoftDeleteMessage(ctx, msgID)
}

func (u *privateChatUsecase) MarkConversationRead(ctx context.Context, userID, convID domain.UUID) error {
	// Get last message in conversation to mark up to
	messages, err := u.repo.ListMessages(ctx, convID, nil, nil, 1)
	if err != nil || len(messages) == 0 {
		return err
	}

	lastMsgID := messages[0].ID
	err = u.repo.MarkAsRead(ctx, userID, convID, lastMsgID)
	if err != nil {
		return err
	}

	// Reset Redis unread count
	unreadKey := fmt.Sprintf("chat:unread:%s:%s", convID, userID)
	_ = u.redis.Del(ctx, unreadKey)
	return nil
}

// Reactions
func (u *privateChatUsecase) ToggleReaction(ctx context.Context, userID, messageID domain.UUID, emoji string) error {
	if !emojiWhitelist[emoji] {
		return errors.New("emoji not in whitelist")
	}

	// Check if already reacted
	reactions, err := u.repo.GetReactionsByMessageID(ctx, messageID)
	if err != nil {
		return err
	}

	var existing *domain.MessageReaction
	userReactionCount := 0
	for _, r := range reactions {
		if r.UserID == userID {
			userReactionCount++
			if r.Emoji == emoji {
				existing = r
			}
		}
	}

	if existing != nil {
		// Unreact
		err = u.repo.DeleteReaction(ctx, messageID, userID, emoji)
	} else {
		if userReactionCount >= 3 {
			return errors.New("max 3 reactions per message exceeded")
		}
		// React
		err = u.repo.AddReaction(ctx, &domain.MessageReaction{
			ID:        domain.NewUUIDv7(),
			MessageID: messageID,
			UserID:    userID,
			Emoji:     emoji,
		})
	}

	// TODO: Broadcast via WebSocket
	return err
}

func (u *privateChatUsecase) GetReactions(ctx context.Context, messageID domain.UUID) (map[string]interface{}, error) {
	reactions, err := u.repo.GetReactionsByMessageID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	// Aggregation
	agg := make(map[string]interface{})
	counts := make(map[string]int)
	for _, r := range reactions {
		counts[r.Emoji]++
	}
	agg["counts"] = counts
	agg["total"] = len(reactions)
	return agg, nil
}

// Disappearing Messages
func (u *privateChatUsecase) MarkAsViewed(ctx context.Context, viewerID, messageID domain.UUID) error {
	// TODO: implement logic for view_once and 7s timer
	return u.repo.TrackView(ctx, &domain.MessageView{
		ID:        domain.NewUUIDv7(),
		MessageID: messageID,
		ViewerID:  viewerID,
	})
}

func (u *privateChatUsecase) NotifyScreenshot(ctx context.Context, userID, conversationID domain.UUID) error {
	// TODO: increment screenshot counter in Redis, broadcast alert
	return nil
}

func (u *privateChatUsecase) ProcessExpiredMessages(ctx context.Context) error {
	messages, err := u.repo.ListExpiredMessages(ctx, time.Now())
	if err != nil {
		return err
	}

	for _, m := range messages {
		// Broadcast expiration
		// Soft delete
		u.repo.UpdateMessageDisappear(ctx, m.ID, nil) // Mark as expired
	}
	return nil
}


func (u *privateChatUsecase) UpdateSettings(ctx context.Context, userID, convID domain.UUID, settings map[string]interface{}) error {
	// Filter allowed settings
	allowed := make(map[string]interface{})
	keys := []string{"is_muted", "is_archived", "is_pinned"}
	for _, k := range keys {
		if v, ok := settings[k]; ok {
			allowed[k] = v
		}
	}

	return u.repo.UpdateConversationSettings(ctx, userID, convID, allowed)
}

func (u *privateChatUsecase) BlockUser(ctx context.Context, blockerID, blockedID domain.UUID, reason string) error {
	return u.repo.BlockUser(ctx, blockerID, blockedID, reason)
}

func (u *privateChatUsecase) UnblockUser(ctx context.Context, blockerID, blockedID domain.UUID) error {
	return u.repo.UnblockUser(ctx, blockerID, blockedID)
}

func (u *privateChatUsecase) GetBlockedUsers(ctx context.Context, userID domain.UUID) ([]*domain.User, error) {
	return u.repo.ListBlockedUsers(ctx, userID)
}

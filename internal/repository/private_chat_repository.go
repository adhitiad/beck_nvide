package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type privateChatRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPrivateChatRepository creates new private chat repository
func NewPrivateChatRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PrivateChatRepository {
	return &privateChatRepository{
		db:     db,
		logger: logger,
	}
}

func (r *privateChatRepository) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Create conversation
	queryConv := `
		INSERT INTO conversations (id, type, initiator_id, recipient_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, queryConv, conv.ID, conv.Type, conv.InitiatorID, conv.RecipientID).Scan(&conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		r.logger.Error("Failed to create conversation", zap.Error(err))
		return err
	}

	// Add participants
	queryPart := `
		INSERT INTO conversation_participants (id, conversation_id, user_id, joined_at, updated_at)
		VALUES (uuid_generate_v7(), $1, $2, NOW(), NOW()),
		       (uuid_generate_v7(), $1, $3, NOW(), NOW())
	`
	_, err = tx.Exec(ctx, queryPart, conv.ID, conv.InitiatorID, conv.RecipientID)
	if err != nil {
		r.logger.Error("Failed to add participants", zap.Error(err))
		return err
	}

	return tx.Commit(ctx)
}

func (r *privateChatRepository) GetConversationByID(ctx context.Context, id domain.UUID) (*domain.Conversation, error) {
	query := `
		SELECT id, type, initiator_id, recipient_id, last_message_id, last_message_at, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`
	conv := &domain.Conversation{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.Type, &conv.InitiatorID, &conv.RecipientID,
		&conv.LastMessageID, &conv.LastMessageAt, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return conv, nil
}

func (r *privateChatRepository) GetConversationByParticipants(ctx context.Context, user1, user2 domain.UUID) (*domain.Conversation, error) {
	query := `
		SELECT id, type, initiator_id, recipient_id, last_message_id, last_message_at, created_at, updated_at
		FROM conversations
		WHERE (initiator_id = $1 AND recipient_id = $2) OR (initiator_id = $2 AND recipient_id = $1)
		LIMIT 1
	`
	conv := &domain.Conversation{}
	err := r.db.QueryRow(ctx, query, user1, user2).Scan(
		&conv.ID, &conv.Type, &conv.InitiatorID, &conv.RecipientID,
		&conv.LastMessageID, &conv.LastMessageAt, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return conv, nil
}

func (r *privateChatRepository) ListConversations(ctx context.Context, userID domain.UUID, cursorTime *time.Time, cursorID *domain.UUID, limit int) ([]*domain.Conversation, error) {
	query := `
		SELECT c.id, c.type, c.initiator_id, c.recipient_id, c.last_message_id, c.last_message_at, c.created_at, c.updated_at,
		       cp.unread_count, cp.is_muted, cp.is_archived, cp.is_pinned,
		       u.username, u.avatar_url
		FROM conversations c
		JOIN conversation_participants cp ON c.id = cp.conversation_id
		JOIN users u ON (CASE WHEN c.initiator_id = $1 THEN c.recipient_id ELSE c.initiator_id END) = u.id
		WHERE cp.user_id = $1 AND cp.is_deleted = false
	`
	args := []interface{}{userID}
	argIdx := 2

	if cursorTime != nil && cursorID != nil {
		query += fmt.Sprintf(" AND (c.last_message_at < $%d OR (c.last_message_at = $%d AND c.id < $%d))", argIdx, argIdx+1, argIdx+2)
		args = append(args, *cursorTime, *cursorTime, *cursorID)
		argIdx += 3
	}

	query += " ORDER BY cp.is_pinned DESC, c.last_message_at DESC, c.id DESC LIMIT $" + fmt.Sprint(argIdx)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]*domain.Conversation, 0)
	for rows.Next() {
		conv := &domain.Conversation{}
		part := domain.ConversationParticipant{}
		otherUser := &domain.User{}
		err := rows.Scan(
			&conv.ID, &conv.Type, &conv.InitiatorID, &conv.RecipientID, &conv.LastMessageID, &conv.LastMessageAt, &conv.CreatedAt, &conv.UpdatedAt,
			&part.UnreadCount, &part.IsMuted, &part.IsArchived, &part.IsPinned,
			&otherUser.Username, &otherUser.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		conv.Participants = []domain.ConversationParticipant{part}
		conv.Recipient = otherUser
		conversations = append(conversations, conv)
	}
	return conversations, nil
}

func (r *privateChatRepository) UpdateConversationSettings(ctx context.Context, userID, convID domain.UUID, settings map[string]interface{}) error {
	if len(settings) == 0 {
		return nil
	}

	query := "UPDATE conversation_participants SET "
	args := []interface{}{userID, convID}
	idx := 3
	for k, v := range settings {
		query += fmt.Sprintf("%s = $%d, ", k, idx)
		args = append(args, v)
		idx++
	}
	query = query[:len(query)-2] // Remove trailing comma
	query += " WHERE user_id = $1 AND conversation_id = $2"

	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *privateChatRepository) DeleteConversation(ctx context.Context, userID, convID domain.UUID) error {
	query := `UPDATE conversation_participants SET is_deleted = true WHERE user_id = $1 AND conversation_id = $2`
	_, err := r.db.Exec(ctx, query, userID, convID)
	return err
}

func (r *privateChatRepository) CreateMessage(ctx context.Context, msg *domain.PrivateMessage) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert message
	queryMsg := `
		INSERT INTO messages (
			id, conversation_id, sender_id, type, content, metadata, reply_to_message_id,
			is_encrypted, disappear_mode, disappear_at, viewed_at, is_screenshot_detected,
			is_expired, is_forwarded, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err = tx.QueryRow(ctx, queryMsg,
		msg.ID, msg.ConversationID, msg.SenderID, msg.Type, msg.Content, msg.Metadata, msg.ReplyToMessageID,
		msg.IsEncrypted, msg.DisappearMode, msg.DisappearAt, msg.ViewedAt, msg.IsScreenshot,
		msg.IsExpired, msg.IsForwarded,
	).Scan(&msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		return err
	}

	// Update conversation last message
	queryConv := `
		UPDATE conversations 
		SET last_message_id = $1, last_message_at = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err = tx.Exec(ctx, queryConv, msg.ID, msg.CreatedAt, msg.ConversationID)
	if err != nil {
		return err
	}

	// Increment unread count for recipient
	queryUnread := `
		UPDATE conversation_participants 
		SET unread_count = unread_count + 1, is_deleted = false, updated_at = NOW()
		WHERE conversation_id = $1 AND user_id != $2
	`
	_, err = tx.Exec(ctx, queryUnread, msg.ConversationID, msg.SenderID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *privateChatRepository) GetMessageByID(ctx context.Context, id domain.UUID) (*domain.PrivateMessage, error) {
	query := `
		SELECT id, conversation_id, sender_id, type, content, metadata, reply_to_message_id, is_edited, is_deleted,
		       is_encrypted, disappear_mode, disappear_at, viewed_at, is_screenshot_detected, is_expired, is_forwarded,
		       created_at, updated_at
		FROM messages
		WHERE id = $1
	`
	msg := &domain.PrivateMessage{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Type, &msg.Content, &msg.Metadata, &msg.ReplyToMessageID,
		&msg.IsEdited, &msg.IsDeleted, &msg.IsEncrypted, &msg.DisappearMode, &msg.DisappearAt, &msg.ViewedAt,
		&msg.IsScreenshot, &msg.IsExpired, &msg.IsForwarded, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *privateChatRepository) ListMessages(ctx context.Context, convID domain.UUID, cursorTime *time.Time, cursorID *domain.UUID, limit int) ([]*domain.PrivateMessage, error) {
	query := `
		SELECT id, conversation_id, sender_id, type, content, metadata, reply_to_message_id, is_edited, is_deleted,
		       is_encrypted, disappear_mode, disappear_at, viewed_at, is_screenshot_detected, is_expired, is_forwarded,
		       created_at, updated_at
		FROM messages
		WHERE conversation_id = $1 AND is_expired = false
	`
	args := []interface{}{convID}
	argIdx := 2

	if cursorTime != nil && cursorID != nil {
		query += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", argIdx, argIdx+1, argIdx+2)
		args = append(args, *cursorTime, *cursorTime, *cursorID)
		argIdx += 3
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT $" + fmt.Sprint(argIdx)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]*domain.PrivateMessage, 0)
	for rows.Next() {
		msg := &domain.PrivateMessage{}
		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Type, &msg.Content, &msg.Metadata, &msg.ReplyToMessageID,
			&msg.IsEdited, &msg.IsDeleted, &msg.IsEncrypted, &msg.DisappearMode, &msg.DisappearAt, &msg.ViewedAt,
			&msg.IsScreenshot, &msg.IsExpired, &msg.IsForwarded, &msg.CreatedAt, &msg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (r *privateChatRepository) UpdateMessage(ctx context.Context, msg *domain.PrivateMessage) error {
	query := `
		UPDATE messages 
		SET content = $1, is_edited = $2, is_deleted = $3, is_expired = $4,
		    disappear_at = $5, viewed_at = $6, is_screenshot_detected = $7,
		    is_forwarded = $8, updated_at = NOW()
		WHERE id = $9
	`
	_, err := r.db.Exec(ctx, query, msg.Content, msg.IsEdited, msg.IsDeleted, msg.IsExpired, msg.DisappearAt, msg.ViewedAt, msg.IsScreenshot, msg.IsForwarded, msg.ID)
	return err
}

func (r *privateChatRepository) SoftDeleteMessage(ctx context.Context, msgID domain.UUID) error {
	query := `UPDATE messages SET is_deleted = true, content = NULL, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, msgID)
	return err
}

func (r *privateChatRepository) MarkAsRead(ctx context.Context, userID, convID domain.UUID, lastMessageID domain.UUID) error {
	query := `
		UPDATE conversation_participants 
		SET unread_count = 0, last_read_message_id = $1, updated_at = NOW()
		WHERE user_id = $2 AND conversation_id = $3
	`
	_, err := r.db.Exec(ctx, query, lastMessageID, userID, convID)
	return err
}

func (r *privateChatRepository) UpdateMessageStatus(ctx context.Context, msgID, userID domain.UUID, status string) error {
	query := `
		INSERT INTO message_status (id, message_id, user_id, status, updated_at)
		VALUES (uuid_generate_v7(), $1, $2, $3, NOW())
		ON CONFLICT (message_id, user_id, status) DO UPDATE SET updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, msgID, userID, status)
	return err
}

func (r *privateChatRepository) BatchUpdateReadStatus(ctx context.Context, convID, userID domain.UUID, lastMessageID domain.UUID) error {
	// Mark all messages in conversation before lastMessageID as read for this user
	// Note: In 1-on-1, this usually means messages sent by the OTHER user
	query := `
		INSERT INTO message_status (id, message_id, user_id, status, updated_at)
		SELECT uuid_generate_v7(), m.id, $1, 'read', NOW()
		FROM messages m
		WHERE m.conversation_id = $2 AND m.sender_id != $1 AND m.created_at <= (SELECT created_at FROM messages WHERE id = $3)
		ON CONFLICT (message_id, user_id, status) DO UPDATE SET updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, userID, convID, lastMessageID)
	return err
}

func (r *privateChatRepository) CreateAttachment(ctx context.Context, att *domain.MessageAttachment) error {
	query := `
		INSERT INTO message_attachments (id, message_id, file_name, file_url, file_type, file_size, width, height, duration, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`
	_, err := r.db.Exec(ctx, query,
		att.ID, att.MessageID, att.FileName, att.FileURL,
		att.FileType, att.FileSize, att.Width, att.Height,
		att.Duration,
	)
	return err
}

// Reactions
func (r *privateChatRepository) AddReaction(ctx context.Context, reaction *domain.MessageReaction) error {
	query := `INSERT INTO message_reactions (id, message_id, user_id, emoji, created_at) VALUES ($1, $2, $3, $4, NOW())`
	_, err := r.db.Exec(ctx, query, reaction.ID, reaction.MessageID, reaction.UserID, reaction.Emoji)
	return err
}

func (r *privateChatRepository) DeleteReaction(ctx context.Context, messageID, userID domain.UUID, emoji string) error {
	query := `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3`
	_, err := r.db.Exec(ctx, query, messageID, userID, emoji)
	return err
}

func (r *privateChatRepository) GetReactionsByMessageID(ctx context.Context, messageID domain.UUID) ([]*domain.MessageReaction, error) {
	query := `SELECT id, message_id, user_id, emoji, created_at FROM message_reactions WHERE message_id = $1`
	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.MessageReaction
	for rows.Next() {
		re := &domain.MessageReaction{}
		err := rows.Scan(&re.ID, &re.MessageID, &re.UserID, &re.Emoji, &re.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, re)
	}
	return result, nil
}

func (r *privateChatRepository) DeleteAllReactions(ctx context.Context, messageID, userID domain.UUID) error {
	query := `DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, messageID, userID)
	return err
}

// Disappearing Messages
func (r *privateChatRepository) TrackView(ctx context.Context, view *domain.MessageView) error {
	query := `INSERT INTO message_views (id, message_id, viewer_id, viewed_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT DO NOTHING`
	_, err := r.db.Exec(ctx, query, view.ID, view.MessageID, view.ViewerID)
	return err
}

func (r *privateChatRepository) UpdateMessageDisappear(ctx context.Context, messageID domain.UUID, disappearAt *time.Time) error {
	query := `UPDATE messages SET disappear_at = $1, viewed_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, disappearAt, messageID)
	return err
}

func (r *privateChatRepository) ListExpiredMessages(ctx context.Context, now time.Time) ([]*domain.PrivateMessage, error) {
	query := `SELECT id, conversation_id, sender_id, type FROM messages WHERE disappear_at <= $1 AND is_expired = false`
	rows, err := r.db.Query(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.PrivateMessage
	for rows.Next() {
		m := &domain.PrivateMessage{}
		err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Type)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}

func (r *privateChatRepository) HardDeleteExpired(ctx context.Context, threshold time.Time) error {
	query := `DELETE FROM messages WHERE is_expired = true AND updated_at < $1`
	_, err := r.db.Exec(ctx, query, threshold)
	return err
}

func (r *privateChatRepository) BlockUser(ctx context.Context, blockerID, blockedID domain.UUID, reason string) error {
	query := `
		INSERT INTO user_blocks (id, blocker_id, blocked_id, reason, created_at)
		VALUES (uuid_generate_v7(), $1, $2, $3, NOW())
		ON CONFLICT (blocker_id, blocked_id) DO UPDATE SET reason = $3, created_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, blockerID, blockedID, reason)
	return err
}

func (r *privateChatRepository) UnblockUser(ctx context.Context, blockerID, blockedID domain.UUID) error {
	query := `DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2`
	_, err := r.db.Exec(ctx, query, blockerID, blockedID)
	return err
}

func (r *privateChatRepository) IsBlocked(ctx context.Context, user1, user2 domain.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM user_blocks WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1))`
	err := r.db.QueryRow(ctx, query, user1, user2).Scan(&exists)
	return exists, err
}

func (r *privateChatRepository) ListBlockedUsers(ctx context.Context, blockerID domain.UUID) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.avatar_url
		FROM users u
		JOIN user_blocks b ON u.id = b.blocked_id
		WHERE b.blocker_id = $1
	`
	rows, err := r.db.Query(ctx, query, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(&user.ID, &user.Username, &user.AvatarURL)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *privateChatRepository) MuteUser(ctx context.Context, blockerID, blockedID domain.UUID, expiresAt *time.Time) error {
	query := `
		INSERT INTO user_mutes (id, muter_id, muted_id, expires_at, created_at)
		VALUES (uuid_generate_v7(), $1, $2, $3, NOW())
		ON CONFLICT (muter_id, muted_id) DO UPDATE SET expires_at = $3, created_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, blockerID, blockedID, expiresAt)
	return err
}

func (r *privateChatRepository) UnmuteUser(ctx context.Context, blockerID, blockedID domain.UUID) error {
	query := `DELETE FROM user_mutes WHERE muter_id = $1 AND muted_id = $2`
	_, err := r.db.Exec(ctx, query, blockerID, blockedID)
	return err
}

func (r *privateChatRepository) IsMuted(ctx context.Context, user1, user2 domain.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_mutes 
			WHERE muter_id = $1 AND muted_id = $2 AND (expires_at IS NULL OR expires_at > NOW())
		)
	`
	err := r.db.QueryRow(ctx, query, user1, user2).Scan(&exists)
	return exists, err
}

func (r *privateChatRepository) ListMutedUsers(ctx context.Context, userID domain.UUID) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.avatar_url
		FROM users u
		JOIN user_mutes m ON u.id = m.muted_id
		WHERE m.muter_id = $1 AND (m.expires_at IS NULL OR m.expires_at > NOW())
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user := &domain.User{}
		err := rows.Scan(&user.ID, &user.Username, &user.AvatarURL)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *privateChatRepository) SaveE2EEKey(ctx context.Context, key *domain.UserE2EEKey) error {
	query := `
		INSERT INTO user_e2ee_keys (user_id, public_key, key_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET public_key = $2, key_type = $3, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, key.UserID, key.PublicKey, key.KeyType)
	return err
}

func (r *privateChatRepository) GetE2EEKey(ctx context.Context, userID domain.UUID) (*domain.UserE2EEKey, error) {
	query := `
		SELECT user_id, public_key, key_type, created_at, updated_at
		FROM user_e2ee_keys
		WHERE user_id = $1
	`
	key := &domain.UserE2EEKey{}
	err := r.db.QueryRow(ctx, query, userID).Scan(&key.UserID, &key.PublicKey, &key.KeyType, &key.CreatedAt, &key.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return key, nil
}

func (r *privateChatRepository) UpdateUserPrivacySettings(ctx context.Context, userID domain.UUID, isPrivate, isIncognito bool) error {
	query := `
		UPDATE users
		SET is_private_profile = $2, is_incognito = $3, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, userID, isPrivate, isIncognito)
	return err
}

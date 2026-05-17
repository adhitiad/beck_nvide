package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type moderationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewModerationRepository(db *pgxpool.Pool, logger *zap.Logger) domain.ModerationRepository {
	return &moderationRepository{
		db:     db,
		logger: logger,
	}
}

// Rules CRUD
func (r *moderationRepository) CreateRule(ctx context.Context, rule *domain.ModerationRule) error {
	query := `
		INSERT INTO moderation_rules (
			id, rule_code, name, category, condition_type, threshold, time_window_seconds,
			action, action_duration_seconds, escalation_rule_id, max_strikes, applies_to,
			is_active, priority, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
		)
	`
	_, err := r.db.Exec(ctx, query,
		rule.ID, rule.RuleCode, rule.Name, rule.Category, rule.ConditionType, rule.Threshold, rule.TimeWindowSeconds,
		rule.Action, rule.ActionDurationSeconds, rule.EscalationRuleID, rule.MaxStrikes, rule.AppliesTo,
		rule.IsActive, rule.Priority,
	)
	if err != nil {
		r.logger.Error("Failed to create moderation rule", zap.Error(err))
		return err
	}
	return nil
}

func (r *moderationRepository) UpdateRule(ctx context.Context, rule *domain.ModerationRule) error {
	query := `
		UPDATE moderation_rules SET
			rule_code = $1, name = $2, category = $3, condition_type = $4, threshold = $5,
			time_window_seconds = $6, action = $7, action_duration_seconds = $8,
			escalation_rule_id = $9, max_strikes = $10, applies_to = $11,
			is_active = $12, priority = $13, updated_at = NOW()
		WHERE id = $14
	`
	_, err := r.db.Exec(ctx, query,
		rule.RuleCode, rule.Name, rule.Category, rule.ConditionType, rule.Threshold,
		rule.TimeWindowSeconds, rule.Action, rule.ActionDurationSeconds,
		rule.EscalationRuleID, rule.MaxStrikes, rule.AppliesTo,
		rule.IsActive, rule.Priority, rule.ID,
	)
	if err != nil {
		r.logger.Error("Failed to update moderation rule", zap.Error(err))
		return err
	}
	return nil
}

func (r *moderationRepository) GetRuleByID(ctx context.Context, id domain.UUID) (*domain.ModerationRule, error) {
	query := `
		SELECT id, rule_code, name, category, condition_type, threshold, time_window_seconds,
		       action, action_duration_seconds, escalation_rule_id, max_strikes, applies_to,
		       is_active, priority, created_at, updated_at
		FROM moderation_rules WHERE id = $1
	`
	rule := &domain.ModerationRule{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rule.ID, &rule.RuleCode, &rule.Name, &rule.Category, &rule.ConditionType, &rule.Threshold, &rule.TimeWindowSeconds,
		&rule.Action, &rule.ActionDurationSeconds, &rule.EscalationRuleID, &rule.MaxStrikes, &rule.AppliesTo,
		&rule.IsActive, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rule, nil
}

func (r *moderationRepository) GetRuleByCode(ctx context.Context, code string) (*domain.ModerationRule, error) {
	query := `
		SELECT id, rule_code, name, category, condition_type, threshold, time_window_seconds,
		       action, action_duration_seconds, escalation_rule_id, max_strikes, applies_to,
		       is_active, priority, created_at, updated_at
		FROM moderation_rules WHERE rule_code = $1
	`
	rule := &domain.ModerationRule{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&rule.ID, &rule.RuleCode, &rule.Name, &rule.Category, &rule.ConditionType, &rule.Threshold, &rule.TimeWindowSeconds,
		&rule.Action, &rule.ActionDurationSeconds, &rule.EscalationRuleID, &rule.MaxStrikes, &rule.AppliesTo,
		&rule.IsActive, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rule, nil
}

func (r *moderationRepository) ListRules(ctx context.Context) ([]*domain.ModerationRule, error) {
	query := `
		SELECT id, rule_code, name, category, condition_type, threshold, time_window_seconds,
		       action, action_duration_seconds, escalation_rule_id, max_strikes, applies_to,
		       is_active, priority, created_at, updated_at
		FROM moderation_rules ORDER BY priority ASC, created_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.ModerationRule
	for rows.Next() {
		rule := &domain.ModerationRule{}
		err = rows.Scan(
			&rule.ID, &rule.RuleCode, &rule.Name, &rule.Category, &rule.ConditionType, &rule.Threshold, &rule.TimeWindowSeconds,
			&rule.Action, &rule.ActionDurationSeconds, &rule.EscalationRuleID, &rule.MaxStrikes, &rule.AppliesTo,
			&rule.IsActive, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func (r *moderationRepository) GetActiveRulesOrdered(ctx context.Context) ([]*domain.ModerationRule, error) {
	query := `
		SELECT id, rule_code, name, category, condition_type, threshold, time_window_seconds,
		       action, action_duration_seconds, escalation_rule_id, max_strikes, applies_to,
		       is_active, priority, created_at, updated_at
		FROM moderation_rules WHERE is_active = true ORDER BY priority ASC, created_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*domain.ModerationRule
	for rows.Next() {
		rule := &domain.ModerationRule{}
		err = rows.Scan(
			&rule.ID, &rule.RuleCode, &rule.Name, &rule.Category, &rule.ConditionType, &rule.Threshold, &rule.TimeWindowSeconds,
			&rule.Action, &rule.ActionDurationSeconds, &rule.EscalationRuleID, &rule.MaxStrikes, &rule.AppliesTo,
			&rule.IsActive, &rule.Priority, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// User State
func (r *moderationRepository) GetUserModerationState(ctx context.Context, userID domain.UUID) (*domain.UserModerationState, error) {
	query := `
		SELECT id, user_id, total_strikes, current_ban_level, is_muted, muted_until, is_banned, banned_until,
		       ban_reason, last_strike_at, last_strike_rule_id, consecutive_same_rule_count, suspected_bot_score,
		       device_fingerprint_hash, ip_cluster_id, created_at, updated_at
		FROM user_moderation_state WHERE user_id = $1
	`
	state := &domain.UserModerationState{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&state.ID, &state.UserID, &state.TotalStrikes, &state.CurrentBanLevel, &state.IsMuted, &state.MutedUntil,
		&state.IsBanned, &state.BannedUntil, &state.BanReason, &state.LastStrikeAt, &state.LastStrikeRuleID,
		&state.ConsecutiveSameRuleCount, &state.SuspectedBotScore, &state.DeviceFingerprintHash, &state.IPClusterID,
		&state.CreatedAt, &state.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return state, nil
}

func (r *moderationRepository) SaveUserModerationState(ctx context.Context, state *domain.UserModerationState) error {
	query := `
		INSERT INTO user_moderation_state (
			id, user_id, total_strikes, current_ban_level, is_muted, muted_until, is_banned, banned_until,
			ban_reason, last_strike_at, last_strike_rule_id, consecutive_same_rule_count, suspected_bot_score,
			device_fingerprint_hash, ip_cluster_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			total_strikes = $3, current_ban_level = $4, is_muted = $5, muted_until = $6,
			is_banned = $7, banned_until = $8, ban_reason = $9, last_strike_at = $10,
			last_strike_rule_id = $11, consecutive_same_rule_count = $12, suspected_bot_score = $13,
			device_fingerprint_hash = $14, ip_cluster_id = $15, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query,
		state.ID, state.UserID, state.TotalStrikes, state.CurrentBanLevel, state.IsMuted, state.MutedUntil,
		state.IsBanned, state.BannedUntil, state.BanReason, state.LastStrikeAt, state.LastStrikeRuleID,
		state.ConsecutiveSameRuleCount, state.SuspectedBotScore, state.DeviceFingerprintHash, state.IPClusterID,
	)
	if err != nil {
		r.logger.Error("Failed to save user moderation state", zap.Error(err))
		return err
	}
	return nil
}

// Logs
func (r *moderationRepository) LogModerationAction(ctx context.Context, log *domain.ModerationLog) error {
	query := `
		INSERT INTO moderation_logs (
			id, user_id, stream_id, conversation_id, rule_id, trigger_type, evidence_type,
			evidence_content, evidence_metadata, action_taken, action_duration_seconds,
			action_executed_at, action_executed_by, related_message_id, related_image_url,
			is_appealed, appeal_status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), $12, $13, $14, $15, $16, NOW()
		)
	`
	_, err := r.db.Exec(ctx, query,
		log.ID, log.UserID, log.StreamID, log.ConversationID, log.RuleID, log.TriggerType, log.EvidenceType,
		log.EvidenceContent, log.EvidenceMetadata, log.ActionTaken, log.ActionDurationSeconds,
		log.ActionExecutedBy, log.RelatedMessageID, log.RelatedImageURL, log.IsAppealed, log.AppealStatus,
	)
	if err != nil {
		r.logger.Error("Failed to log moderation action", zap.Error(err))
		return err
	}
	return nil
}

func (r *moderationRepository) ListModerationLogs(ctx context.Context, userID *domain.UUID, streamID *domain.UUID, action *string, limit, offset int) ([]*domain.ModerationLog, error) {
	query := `
		SELECT id, user_id, stream_id, conversation_id, rule_id, trigger_type, evidence_type,
		       evidence_content, evidence_metadata, action_taken, action_duration_seconds,
		       action_executed_at, action_executed_by, related_message_id, related_image_url,
		       is_appealed, appeal_status, created_at
		FROM moderation_logs
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND ($2::uuid IS NULL OR stream_id = $2)
		  AND ($3::text IS NULL OR action_taken = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.db.Query(ctx, query, userID, streamID, action, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.ModerationLog
	for rows.Next() {
		log := &domain.ModerationLog{}
		err = rows.Scan(
			&log.ID, &log.UserID, &log.StreamID, &log.ConversationID, &log.RuleID, &log.TriggerType, &log.EvidenceType,
			&log.EvidenceContent, &log.EvidenceMetadata, &log.ActionTaken, &log.ActionDurationSeconds,
			&log.ActionExecutedAt, &log.ActionExecutedBy, &log.RelatedMessageID, &log.RelatedImageURL,
			&log.IsAppealed, &log.AppealStatus, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func (r *moderationRepository) GetActiveBans(ctx context.Context) ([]*domain.UserModerationState, error) {
	query := `
		SELECT id, user_id, total_strikes, current_ban_level, is_muted, muted_until, is_banned, banned_until,
		       ban_reason, last_strike_at, last_strike_rule_id, consecutive_same_rule_count, suspected_bot_score,
		       device_fingerprint_hash, ip_cluster_id, created_at, updated_at
		FROM user_moderation_state
		WHERE is_banned = true OR is_muted = true
		ORDER BY updated_at DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []*domain.UserModerationState
	for rows.Next() {
		state := &domain.UserModerationState{}
		err = rows.Scan(
			&state.ID, &state.UserID, &state.TotalStrikes, &state.CurrentBanLevel, &state.IsMuted, &state.MutedUntil,
			&state.IsBanned, &state.BannedUntil, &state.BanReason, &state.LastStrikeAt, &state.LastStrikeRuleID,
			&state.ConsecutiveSameRuleCount, &state.SuspectedBotScore, &state.DeviceFingerprintHash, &state.IPClusterID,
			&state.CreatedAt, &state.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, nil
}

func (r *moderationRepository) GetModerationLogByID(ctx context.Context, id domain.UUID) (*domain.ModerationLog, error) {
	query := `
		SELECT id, user_id, stream_id, conversation_id, rule_id, trigger_type, evidence_type,
		       evidence_content, evidence_metadata, action_taken, action_duration_seconds,
		       action_executed_at, action_executed_by, related_message_id, related_image_url,
		       is_appealed, appeal_status, created_at
		FROM moderation_logs WHERE id = $1
	`
	log := &domain.ModerationLog{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&log.ID, &log.UserID, &log.StreamID, &log.ConversationID, &log.RuleID, &log.TriggerType, &log.EvidenceType,
		&log.EvidenceContent, &log.EvidenceMetadata, &log.ActionTaken, &log.ActionDurationSeconds,
		&log.ActionExecutedAt, &log.ActionExecutedBy, &log.RelatedMessageID, &log.RelatedImageURL,
		&log.IsAppealed, &log.AppealStatus, &log.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return log, nil
}

// Image Moderation Queue
func (r *moderationRepository) EnqueueImage(ctx context.Context, q *domain.ImageModerationQueue) error {
	query := `
		INSERT INTO image_moderation_queue (
			id, image_url, source_type, source_id, status, provider, nsfw_score,
			is_nsfw, moderation_labels, action_taken, blurred_url, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
		)
	`
	_, err := r.db.Exec(ctx, query,
		q.ID, q.ImageURL, q.SourceType, q.SourceID, q.Status, q.Provider, q.NSFWScore,
		q.IsNSFW, q.ModerationLabels, q.ActionTaken, q.BlurredURL,
	)
	if err != nil {
		r.logger.Error("Failed to enqueue image for scanning", zap.Error(err))
		return err
	}
	return nil
}

func (r *moderationRepository) GetPendingImages(ctx context.Context) ([]*domain.ImageModerationQueue, error) {
	query := `
		SELECT id, image_url, source_type, source_id, status, provider, nsfw_score,
		       is_nsfw, moderation_labels, action_taken, blurred_url, created_at, completed_at
		FROM image_moderation_queue
		WHERE status = 'queued'
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.ImageModerationQueue
	for rows.Next() {
		q := &domain.ImageModerationQueue{}
		err = rows.Scan(
			&q.ID, &q.ImageURL, &q.SourceType, &q.SourceID, &q.Status, &q.Provider, &q.NSFWScore,
			&q.IsNSFW, &q.ModerationLabels, &q.ActionTaken, &q.BlurredURL, &q.CreatedAt, &q.CompletedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, q)
	}
	return jobs, nil
}

func (r *moderationRepository) GetImageJobByID(ctx context.Context, id domain.UUID) (*domain.ImageModerationQueue, error) {
	query := `
		SELECT id, image_url, source_type, source_id, status, provider, nsfw_score,
		       is_nsfw, moderation_labels, action_taken, blurred_url, created_at, completed_at
		FROM image_moderation_queue WHERE id = $1
	`
	q := &domain.ImageModerationQueue{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&q.ID, &q.ImageURL, &q.SourceType, &q.SourceID, &q.Status, &q.Provider, &q.NSFWScore,
		&q.IsNSFW, &q.ModerationLabels, &q.ActionTaken, &q.BlurredURL, &q.CreatedAt, &q.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return q, nil
}

func (r *moderationRepository) UpdateImageJob(ctx context.Context, q *domain.ImageModerationQueue) error {
	query := `
		UPDATE image_moderation_queue SET
			status = $1, provider = $2, nsfw_score = $3, is_nsfw = $4,
			moderation_labels = $5, action_taken = $6, blurred_url = $7, completed_at = $8
		WHERE id = $9
	`
	_, err := r.db.Exec(ctx, query,
		q.Status, q.Provider, q.NSFWScore, q.IsNSFW, q.ModerationLabels, q.ActionTaken, q.BlurredURL, q.CompletedAt, q.ID,
	)
	if err != nil {
		r.logger.Error("Failed to update image job status", zap.Error(err))
		return err
	}
	return nil
}

// Wordlist
func (r *moderationRepository) GetWordlist(ctx context.Context) ([]*domain.ModerationWordlist, error) {
	query := `SELECT id, word, severity_level, language, is_regex, created_at FROM moderation_wordlist ORDER BY word ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []*domain.ModerationWordlist
	for rows.Next() {
		w := &domain.ModerationWordlist{}
		err = rows.Scan(&w.ID, &w.Word, &w.SeverityLevel, &w.Language, &w.IsRegex, &w.CreatedAt)
		if err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, nil
}

func (r *moderationRepository) AddWord(ctx context.Context, w *domain.ModerationWordlist) error {
	query := `
		INSERT INTO moderation_wordlist (id, word, severity_level, language, is_regex, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (word) DO UPDATE SET severity_level = $3, language = $4, is_regex = $5
	`
	_, err := r.db.Exec(ctx, query, w.ID, w.Word, w.SeverityLevel, w.Language, w.IsRegex)
	if err != nil {
		r.logger.Error("Failed to add word to filter wordlist", zap.Error(err))
		return err
	}
	return nil
}

func (r *moderationRepository) DeleteWord(ctx context.Context, word string) error {
	query := `DELETE FROM moderation_wordlist WHERE word = $1`
	_, err := r.db.Exec(ctx, query, word)
	if err != nil {
		return err
	}
	return nil
}

// Gift hold logic
func (r *moderationRepository) HoldGiftTransaction(ctx context.Context, txID domain.UUID) error {
	query := `UPDATE gift_transactions SET status = 'held' WHERE id = $1`
	_, err := r.db.Exec(ctx, query, txID)
	return err
}

func (r *moderationRepository) ReleaseGiftTransaction(ctx context.Context, txID domain.UUID) error {
	query := `UPDATE gift_transactions SET status = 'completed' WHERE id = $1`
	_, err := r.db.Exec(ctx, query, txID)
	return err
}

func (r *moderationRepository) GetStreamGiftTransactionsInWindow(ctx context.Context, userID domain.UUID, window time.Duration) ([]*domain.GiftTransaction, error) {
	thresholdTime := time.Now().Add(-window)
	query := `
		SELECT id, stream_id, sender_id, receiver_id, gift_id, quantity, total_price, agency_id, agency_commission, host_earning, platform_fee, created_at
		FROM gift_transactions
		WHERE sender_id = $1 AND created_at >= $2
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID, thresholdTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.GiftTransaction
	for rows.Next() {
		tx := &domain.GiftTransaction{}
		err = rows.Scan(
			&tx.ID, &tx.StreamID, &tx.SenderID, &tx.ReceiverID, &tx.GiftID, &tx.Quantity, &tx.TotalPrice,
			&tx.AgencyID, &tx.AgencyCommission, &tx.HostEarning, &tx.PlatformFee, &tx.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (r *moderationRepository) GetUsernameByID(ctx context.Context, userID domain.UUID) (string, error) {
	var username string
	query := `SELECT username FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, userID).Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}

func (r *moderationRepository) ExistsUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	err := r.db.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *moderationRepository) SaveGiftFraudAlert(ctx context.Context, alert *domain.GiftFraudAlert) error {
	query := `
		INSERT INTO gift_fraud_alerts (
			id, alert_type, primary_user_id, secondary_user_id, total_gift_value_id_r,
			transaction_count, time_window_seconds, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`
	_, err := r.db.Exec(ctx, query,
		alert.ID, alert.AlertType, alert.PrimaryUserID, alert.SecondaryUserID, alert.TotalGiftValueIDR,
		alert.TransactionCount, alert.TimeWindowSeconds, alert.Status,
	)
	return err
}

func (r *moderationRepository) SubmitAppealUpdate(ctx context.Context, logID domain.UUID, updatedEvidenceContent string) error {
	query := `UPDATE moderation_logs SET is_appealed = true, appeal_status = 'pending', evidence_content = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, updatedEvidenceContent, logID)
	return err
}

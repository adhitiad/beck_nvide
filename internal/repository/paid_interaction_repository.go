package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type paidInteractionRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewPaidInteractionRepository(db *pgxpool.Pool, logger *zap.Logger) domain.PaidInteractionRepository {
	return &paidInteractionRepository{
		db:     db,
		logger: logger,
	}
}

func (r *paidInteractionRepository) GetHostCallRate(ctx context.Context, hostID domain.UUID) (*domain.HostCallRate, error) {
	query := `
		SELECT id, host_id, voice_call_rate_idr, video_call_rate_idr, min_duration_seconds, is_enabled, updated_at
		FROM host_call_rates
		WHERE host_id = $1
	`
	rate := &domain.HostCallRate{}
	err := r.db.QueryRow(ctx, query, hostID).Scan(
		&rate.ID, &rate.HostID, &rate.VoiceCallRateIDR, &rate.VideoCallRateIDR,
		&rate.MinDurationSeconds, &rate.IsEnabled, &rate.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rate, nil
}

func (r *paidInteractionRepository) UpsertHostCallRate(ctx context.Context, rate *domain.HostCallRate) error {
	query := `
		INSERT INTO host_call_rates (id, host_id, voice_call_rate_idr, video_call_rate_idr, is_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (host_id) DO UPDATE SET
			voice_call_rate_idr = EXCLUDED.voice_call_rate_idr,
			video_call_rate_idr = EXCLUDED.video_call_rate_idr,
			is_enabled = EXCLUDED.is_enabled,
			updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, rate.ID, rate.HostID, rate.VoiceCallRateIDR, rate.VideoCallRateIDR, rate.IsEnabled)
	return err
}

func (r *paidInteractionRepository) CreatePaidChatUnlock(ctx context.Context, unlock *domain.PaidChatUnlock) error {
	query := `
		INSERT INTO paid_chat_unlocks (id, conversation_id, payer_id, recipient_id, amount_idr, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.Exec(ctx, query, unlock.ID, unlock.ConversationID, unlock.PayerID, unlock.RecipientID, unlock.AmountIDR, unlock.Status)
	return err
}

func (r *paidInteractionRepository) GetPaidChatUnlock(ctx context.Context, convID, payerID domain.UUID) (*domain.PaidChatUnlock, error) {
	query := `
		SELECT id, conversation_id, payer_id, recipient_id, amount_idr, status, created_at
		FROM paid_chat_unlocks
		WHERE conversation_id = $1 AND payer_id = $2
		LIMIT 1
	`
	unlock := &domain.PaidChatUnlock{}
	err := r.db.QueryRow(ctx, query, convID, payerID).Scan(
		&unlock.ID, &unlock.ConversationID, &unlock.PayerID, &unlock.RecipientID,
		&unlock.AmountIDR, &unlock.Status, &unlock.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return unlock, nil
}

func (r *paidInteractionRepository) IsChatUnlocked(ctx context.Context, convID, payerID domain.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM paid_chat_unlocks WHERE conversation_id = $1 AND payer_id = $2 AND status = 'active')`
	err := r.db.QueryRow(ctx, query, convID, payerID).Scan(&exists)
	return exists, err
}

func (r *paidInteractionRepository) ListPendingRefunds(ctx context.Context, threshold time.Time) ([]*domain.PaidChatUnlock, error) {
	// Find active unlocks created before threshold that have no replies from recipient in conversation
	query := `
		SELECT u.id, u.conversation_id, u.payer_id, u.recipient_id, u.amount_idr, u.status, u.created_at
		FROM paid_chat_unlocks u
		WHERE u.status = 'active' AND u.created_at < $1
		AND NOT EXISTS (
			SELECT 1 FROM messages m
			WHERE m.conversation_id = u.conversation_id
			AND m.sender_id = u.recipient_id
			AND m.created_at > u.created_at
		)
	`
	rows, err := r.db.Query(ctx, query, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var unlocks []*domain.PaidChatUnlock
	for rows.Next() {
		u := &domain.PaidChatUnlock{}
		err := rows.Scan(&u.ID, &u.ConversationID, &u.PayerID, &u.RecipientID, &u.AmountIDR, &u.Status, &u.CreatedAt)
		if err != nil {
			return nil, err
		}
		unlocks = append(unlocks, u)
	}
	return unlocks, nil
}

func (r *paidInteractionRepository) UpdateUnlockStatus(ctx context.Context, id domain.UUID, status string) error {
	query := `UPDATE paid_chat_unlocks SET status = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

func (r *paidInteractionRepository) CreateCallSession(ctx context.Context, session *domain.CallSession) error {
	query := `
		INSERT INTO call_sessions (id, host_id, caller_id, type, rate_idr, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.Exec(ctx, query, session.ID, session.HostID, session.CallerID, session.Type, session.RateIDR, session.Status)
	return err
}

func (r *paidInteractionRepository) GetCallSessionByID(ctx context.Context, id domain.UUID) (*domain.CallSession, error) {
	query := `
		SELECT id, host_id, caller_id, type, rate_idr, status, started_at, ended_at, duration_seconds, total_charge_idr, platform_fee_idr, host_earning_idr, ended_reason, created_at
		FROM call_sessions
		WHERE id = $1
	`
	s := &domain.CallSession{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.HostID, &s.CallerID, &s.Type, &s.RateIDR, &s.Status,
		&s.StartedAt, &s.EndedAt, &s.DurationSeconds, &s.TotalChargeIDR,
		&s.PlatformFeeIDR, &s.HostEarningIDR, &s.EndedReason, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *paidInteractionRepository) UpdateCallSession(ctx context.Context, s *domain.CallSession) error {
	query := `
		UPDATE call_sessions
		SET status = $1, started_at = $2, ended_at = $3, duration_seconds = $4,
		    total_charge_idr = $5, platform_fee_idr = $6, host_earning_idr = $7,
		    ended_reason = $8
		WHERE id = $9
	`
	_, err := r.db.Exec(ctx, query,
		s.Status, s.StartedAt, s.EndedAt, s.DurationSeconds,
		s.TotalChargeIDR, s.PlatformFeeIDR, s.HostEarningIDR,
		s.EndedReason, s.ID,
	)
	return err
}

func (r *paidInteractionRepository) CreateBillingTick(ctx context.Context, tick *domain.CallBillingTick) error {
	query := `
		INSERT INTO call_billing_ticks (id, call_session_id, tick_number, charge_idr, deducted_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err := r.db.Exec(ctx, query, tick.ID, tick.CallSessionID, tick.TickNumber, tick.ChargeIDR)
	return err
}

func (r *paidInteractionRepository) GetCallHistory(ctx context.Context, userID domain.UUID, role string, limit, offset int) ([]*domain.CallSession, error) {
	var query string
	if role == "host" {
		query = `SELECT id, caller_id, host_id, status, created_at FROM call_sessions WHERE host_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	} else {
		query = `SELECT id, caller_id, host_id, status, created_at FROM call_sessions WHERE caller_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	}

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*domain.CallSession
	for rows.Next() {
		s := &domain.CallSession{}
		// Scan simplified for now
		err := rows.Scan(&s.ID, &s.CallerID, &s.HostID, &s.Status, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

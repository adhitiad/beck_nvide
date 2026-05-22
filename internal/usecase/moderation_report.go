package usecase

import (
	"context"
	"fmt"
	"time"

	"nvide-live/internal/domain"
)

// Appeals & Support
func (u *moderationUseCase) SubmitAppeal(ctx context.Context, logID domain.UUID, reason string) error {
	log, err := u.repo.GetModerationLogByID(ctx, logID)
	if err != nil || log == nil {
		return fmt.Errorf("log not found")
	}

	now := time.Now()
	// Mute/kick appeal window: 15 minutes. Ban appeal window: 7 days
	limit := 15 * time.Minute
	if log.ActionTaken == "ban_temp" || log.ActionTaken == "ban_perm" {
		limit = 7 * 24 * time.Hour
	}

	if now.Sub(log.ActionExecutedAt) > limit {
		return fmt.Errorf("appeal window has expired for this action")
	}

	log.IsAppealed = true
	log.AppealStatus = "pending"
	log.EvidenceContent = log.EvidenceContent + "\n[APPEAL REASON]: " + reason

	return u.repo.SubmitAppealUpdate(ctx, logID, log.EvidenceContent)
}

func (u *moderationUseCase) ListLogs(ctx context.Context, userID *domain.UUID, streamID *domain.UUID, action *string, limit, offset int) ([]*domain.ModerationLog, error) {
	return u.repo.ListModerationLogs(ctx, userID, streamID, action, limit, offset)
}

func (u *moderationUseCase) GetActiveBans(ctx context.Context) ([]*domain.UserModerationState, error) {
	return u.repo.GetActiveBans(ctx)
}

func (u *moderationUseCase) ManualOverride(ctx context.Context, adminID domain.UUID, userID domain.UUID, actionType string, reason string) error {
	state, err := u.repo.GetUserModerationState(ctx, userID)
	if err != nil {
		return err
	}

	if state == nil {
		state = &domain.UserModerationState{
			ID:     domain.NewUUID(),
			UserID: userID,
		}
	}

	now := time.Now()

	if actionType == "unmute" {
		state.IsMuted = false
		state.MutedUntil = nil
		_ = u.redis.Del(ctx, fmt.Sprintf("mod:mute:%s", userID))
	} else if actionType == "unban" {
		state.IsBanned = false
		state.BannedUntil = nil
		_ = u.redis.Del(ctx, fmt.Sprintf("jwt:blacklist:%s", userID))
	} else if actionType == "mute" {
		state.IsMuted = true
		until := now.Add(24 * time.Hour) // manual default 24h
		state.MutedUntil = &until
		_ = u.redis.Set(ctx, fmt.Sprintf("mod:mute:%s", userID), "1", 24*time.Hour)
	} else if actionType == "ban_perm" {
		state.IsBanned = true
		state.BanReason = reason
		_ = u.redis.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", userID), "1", 365*24*time.Hour)
	}

	err = u.repo.SaveUserModerationState(ctx, state)
	if err != nil {
		return err
	}

	// Immutable log
	return u.repo.LogModerationAction(ctx, &domain.ModerationLog{
		ID:               domain.NewUUID(),
		UserID:           userID,
		TriggerType:      "manual",
		EvidenceType:     "admin_override",
		EvidenceContent:  reason,
		ActionTaken:      actionType,
		ActionExecutedBy: &adminID,
	})
}

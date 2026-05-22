package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type MissionUseCase struct {
	missionRepo domain.MissionRepository
	walletRepo  domain.WalletRepository
	logger      *zap.Logger
}

func NewMissionUseCase(
	missionRepo domain.MissionRepository,
	walletRepo domain.WalletRepository,
	logger *zap.Logger,
) *MissionUseCase {
	return &MissionUseCase{
		missionRepo: missionRepo,
		walletRepo:  walletRepo,
		logger:      logger,
	}
}

// GetDailyMissions returns today's missions for a user
func (uc *MissionUseCase) GetDailyMissions(ctx context.Context, userID domain.UUID) ([]*domain.UserMission, error) {
	return uc.missionRepo.GetOrCreateUserMissions(ctx, userID, time.Now())
}

// TrackProgress increments mission progress for a specific mission type
func (uc *MissionUseCase) TrackProgress(ctx context.Context, userID domain.UUID, missionType string, delta int) error {
	return uc.missionRepo.IncrementMissionProgress(ctx, userID, missionType, delta, time.Now())
}

// ClaimReward claims the reward for a completed mission
func (uc *MissionUseCase) ClaimReward(ctx context.Context, userID, missionID domain.UUID) error {
	um, err := uc.missionRepo.GetUserMission(ctx, userID, missionID, time.Now())
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "mission not found", err)
	}

	if !um.IsCompleted {
		return domain.NewDomainError(domain.ErrCodeValidation, "mission not completed yet", nil)
	}
	if um.IsClaimed {
		return domain.NewDomainError(domain.ErrCodeConflict, "reward already claimed", nil)
	}

	mission, err := uc.missionRepo.GetMissionByID(ctx, missionID)
	if err != nil {
		return err
	}

	// Apply reward
	switch mission.RewardType {
	case domain.RewardTypeCoin:
		_ = uc.walletRepo.CreditBalance(ctx, userID, mission.RewardValue)
	case domain.RewardTypeEXP:
		// XP is handled by the leveling system (already exists)
	}

	// Mark as claimed
	return uc.missionRepo.ClaimReward(ctx, um.ID)
}

// GetBadges returns all badges for a user
func (uc *MissionUseCase) GetBadges(ctx context.Context, userID domain.UUID) ([]*domain.UserBadge, error) {
	return uc.missionRepo.GetUserBadges(ctx, userID)
}

// CheckAndAwardBadges evaluates badge criteria and awards new badges
func (uc *MissionUseCase) CheckAndAwardBadges(ctx context.Context, userID domain.UUID, achievementKey string, badgeName, icon, description string) error {
	has, _ := uc.missionRepo.HasBadge(ctx, userID, achievementKey)
	if has {
		return nil // Already has this badge
	}

	badge := &domain.UserBadge{
		ID:             domain.NewUUID(),
		UserID:         userID,
		BadgeName:      badgeName,
		BadgeIcon:      icon,
		AchievementKey: achievementKey,
		Description:    description,
	}
	return uc.missionRepo.AwardBadge(ctx, badge)
}

// GetLeaderboard returns leaderboard for a given type and period
func (uc *MissionUseCase) GetLeaderboard(ctx context.Context, lbType, period string, limit int) ([]*domain.LeaderboardEntry, error) {
	date := time.Now()
	return uc.missionRepo.GetLeaderboard(ctx, lbType, period, date, limit)
}

// GetMyRank returns the user's rank on a leaderboard
func (uc *MissionUseCase) GetMyRank(ctx context.Context, userID domain.UUID, lbType, period string) (*domain.LeaderboardEntry, error) {
	return uc.missionRepo.GetUserRank(ctx, lbType, period, userID, time.Now())
}

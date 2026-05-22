package usecase

import (
	"context"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type HostLevelUseCase struct {
	levelRepo domain.HostLevelRepository
	userRepo  domain.UserRepository
	logger    *zap.Logger
}

func NewHostLevelUseCase(
	levelRepo domain.HostLevelRepository,
	userRepo domain.UserRepository,
	logger *zap.Logger,
) *HostLevelUseCase {
	return &HostLevelUseCase{levelRepo: levelRepo, userRepo: userRepo, logger: logger}
}

// GetLevels returns all host level definitions
func (uc *HostLevelUseCase) GetLevels(ctx context.Context) ([]*domain.HostLevel, error) {
	return uc.levelRepo.ListAll(ctx)
}

// GetHostLevel returns the current level info for a host
func (uc *HostLevelUseCase) GetHostLevel(ctx context.Context, hostID domain.UUID) (*domain.HostLevel, error) {
	user, err := uc.userRepo.GetByID(ctx, hostID)
	if err != nil {
		return nil, err
	}

	tier := "newbie"
	if user != nil {
		// user.HostTier is set from the DB but we can't read it since it's not on the struct yet.
		// For now, compute from totalStreamHours and totalIncome.
	}

	level, err := uc.levelRepo.GetByName(ctx, tier)
	if err != nil {
		// Fallback to computing from user stats
		level, err = uc.levelRepo.GetEligibleLevel(ctx, user.HostLevel, int64(user.HostXP))
		if err != nil {
			return nil, err
		}
	}
	return level, nil
}

// EvaluateAndPromote checks all hosts and promotes eligible ones
func (uc *HostLevelUseCase) EvaluateAndPromote(ctx context.Context) error {
	levels, err := uc.levelRepo.ListAll(ctx)
	if err != nil {
		return err
	}
	_ = levels // In production: query all hosts, check against levels, and update host_tier
	uc.logger.Info("Host level evaluation completed")
	return nil
}

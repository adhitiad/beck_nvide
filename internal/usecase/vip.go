package usecase

import (
	"context"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type VIPUseCase struct {
	vipRepo    domain.VIPRepository
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
	redis      *redis.Client
	logger     *zap.Logger
}

func NewVIPUseCase(
	vipRepo domain.VIPRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	redis *redis.Client,
	logger *zap.Logger,
) *VIPUseCase {
	return &VIPUseCase{
		vipRepo:    vipRepo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		redis:      redis,
		logger:     logger,
	}
}

// ListPlans returns all available VIP levels
func (uc *VIPUseCase) ListPlans(ctx context.Context) ([]*domain.VIPLevel, error) {
	return uc.vipRepo.ListLevels(ctx)
}

// Subscribe purchases a VIP subscription for a user
func (uc *VIPUseCase) Subscribe(ctx context.Context, userID domain.UUID, levelName string) (*domain.UserVIP, error) {
	level, err := uc.vipRepo.GetLevelByName(ctx, levelName)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "VIP level not found", err)
	}

	// Check if user already has active VIP of same or higher level
	existing, _ := uc.vipRepo.GetActiveByUserID(ctx, userID)
	if existing != nil && existing.VIPLevel != nil && existing.VIPLevel.SortOrder >= level.SortOrder {
		return nil, domain.NewDomainError(domain.ErrCodeConflict,
			"user already has same or higher VIP level active", nil)
	}

	// Debit wallet
	if err := uc.walletRepo.DebitBalance(ctx, userID, level.Price); err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance for VIP purchase", err)
	}

	// Record transaction
	tx := &domain.Transaction{
		ID:          domain.NewUUID(),
		UserID:      userID,
		Type:        "vip_purchase",
		Amount:      level.Price,
		Currency:    "IDR",
		Status:      domain.TxStatusSuccess,
		ReferenceID: "vip_" + level.Name,
	}
	_ = uc.txRepo.Create(ctx, tx)

	// Create VIP subscription
	now := time.Now()
	uv := &domain.UserVIP{
		ID:         domain.NewUUID(),
		UserID:     userID,
		VIPLevelID: level.ID,
		StartedAt:  now,
		ExpiresAt:  now.AddDate(0, 0, level.DurationDays),
		AutoRenew:  false,
	}

	if err := uc.vipRepo.Subscribe(ctx, uv); err != nil {
		return nil, err
	}

	uv.VIPLevel = level
	uc.logger.Info("User subscribed to VIP",
		zap.String("user_id", string(userID)),
		zap.String("level", level.Name),
	)

	return uv, nil
}

// GetMyVIP returns the active VIP subscription for a user
func (uc *VIPUseCase) GetMyVIP(ctx context.Context, userID domain.UUID) (*domain.UserVIP, error) {
	return uc.vipRepo.GetActiveByUserID(ctx, userID)
}

// GetVIPHistory returns the VIP subscription history for a user
func (uc *VIPUseCase) GetVIPHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.UserVIP, error) {
	return uc.vipRepo.ListByUserID(ctx, userID, limit, offset)
}

// SetAutoRenew toggles auto-renew for a VIP subscription
func (uc *VIPUseCase) SetAutoRenew(ctx context.Context, userID domain.UUID, autoRenew bool) error {
	uv, err := uc.vipRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "no active VIP subscription found", err)
	}
	if uv.UserID != userID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "not your subscription", nil)
	}
	return uc.vipRepo.UpdateAutoRenew(ctx, uv.ID, autoRenew)
}

// GetEmoticons returns available emoticons for a user based on their VIP level
func (uc *VIPUseCase) GetEmoticons(ctx context.Context, userID domain.UUID) ([]*domain.VIPEmoticon, error) {
	uv, err := uc.vipRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		// Non-VIP users get no special emoticons
		return []*domain.VIPEmoticon{}, nil
	}
	return uc.vipRepo.ListEmoticonsByLevel(ctx, uv.VIPLevelID)
}

// GetEntryEffect returns a random entry effect for a user's VIP level
func (uc *VIPUseCase) GetEntryEffect(ctx context.Context, userID domain.UUID) (*domain.EntryEffect, error) {
	uv, err := uc.vipRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil // No VIP, no effect
	}
	return uc.vipRepo.GetRandomEffect(ctx, uv.VIPLevelID)
}

// CheckPrivilege checks if a user has a specific VIP privilege
func (uc *VIPUseCase) CheckPrivilege(ctx context.Context, userID domain.UUID, privilege string) bool {
	uv, err := uc.vipRepo.GetActiveByUserID(ctx, userID)
	if err != nil || uv == nil || uv.VIPLevel == nil {
		return false
	}
	// Privileges stored as JSONB string — simple contains check
	return len(uv.VIPLevel.Privileges) > 0 &&
		contains(uv.VIPLevel.Privileges, `"`+privilege+`":true`)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ProcessExpiredVIP handles auto-renewal for expiring VIP subscriptions
func (uc *VIPUseCase) ProcessExpiredVIP(ctx context.Context) error {
	expiring, err := uc.vipRepo.ListExpiring(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		return err
	}

	for _, uv := range expiring {
		level, err := uc.vipRepo.GetLevelByID(ctx, uv.VIPLevelID)
		if err != nil {
			continue
		}

		// Try auto-renew
		if err := uc.walletRepo.DebitBalance(ctx, uv.UserID, level.Price); err != nil {
			uc.logger.Warn("VIP auto-renew failed (insufficient balance)",
				zap.String("user_id", string(uv.UserID)))
			continue
		}

		newUV := &domain.UserVIP{
			ID:         domain.NewUUID(),
			UserID:     uv.UserID,
			VIPLevelID: uv.VIPLevelID,
			StartedAt:  time.Now(),
			ExpiresAt:  time.Now().AddDate(0, 0, level.DurationDays),
			AutoRenew:  true,
		}
		_ = uc.vipRepo.Subscribe(ctx, newUV)

		uc.logger.Info("VIP auto-renewed",
			zap.String("user_id", string(uv.UserID)),
			zap.String("level", level.Name),
		)
	}
	return nil
}

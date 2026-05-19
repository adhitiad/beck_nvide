package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

var (
	ErrNoActiveSubscription = errors.New("no active VIP AI Clip subscription found")
	ErrQuotaExceeded        = errors.New("VIP AI Clip quota has been fully consumed for this billing cycle")
)

type ClipSubscriptionUseCase struct {
	subRepo     domain.ClipSubscriptionRepository
	userRepo    domain.UserRepository
	walletUC    *WalletUseCase
	logger      *zap.Logger
}

func NewClipSubscriptionUseCase(
	subRepo domain.ClipSubscriptionRepository,
	userRepo domain.UserRepository,
	walletUC *WalletUseCase,
	logger *zap.Logger,
) *ClipSubscriptionUseCase {
	return &ClipSubscriptionUseCase{
		subRepo:  subRepo,
		userRepo: userRepo,
		walletUC: walletUC,
		logger:   logger,
	}
}

// ListPlans lists VIP packages and applies "Promo Host Pertama" if applicable
func (uc *ClipSubscriptionUseCase) ListPlans(ctx context.Context, userID domain.UUID) ([]*domain.ClipSubscriptionPlan, error) {
	plans, err := uc.subRepo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}

	// Determine if "Promo Host Pertama" applies
	isEligibleForPromo := false
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err == nil {
		isHost := user.Role != nil && user.Role.Name == "host"
		if isHost {
			hasSubbed, err := uc.subRepo.HasSubscribedBefore(ctx, userID)
			if err == nil && !hasSubbed {
				isEligibleForPromo = true
			}
		}
	}

	// Apply promotional price to VIP1 if eligible
	for _, p := range plans {
		if p.Name == "VIP1" && isEligibleForPromo {
			p.Price = 15678 // Promo price Rp 15.678
		}
	}

	return plans, nil
}

// Subscribe handles choosing a package, debiting wallet, and activating VIP package
func (uc *ClipSubscriptionUseCase) Subscribe(ctx context.Context, userID domain.UUID, planID domain.UUID) (*domain.ClipSubscription, error) {
	// 1. Fetch Plan details
	plan, err := uc.subRepo.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch User and check eligibility for Promo Host Pertama
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	price := plan.Price
	isHost := user.Role != nil && user.Role.Name == "host"
	if plan.Name == "VIP1" && isHost {
		hasSubbed, err := uc.subRepo.HasSubscribedBefore(ctx, userID)
		if err == nil && !hasSubbed {
			price = 15678 // Promo price Rp 15.678
		}
	}

	// 3. Debit wallet using WalletUseCase
	// This will fail if balance is insufficient
	refID := fmt.Sprintf("sub_%s_%d", plan.Name, time.Now().Unix())
	err = uc.walletUC.DebitWallet(ctx, userID, price, "vip_subscription", refID)
	if err != nil {
		uc.logger.Warn("Failed to purchase VIP subscription due to wallet debit error", 
			zap.String("user_id", userID.String()), 
			zap.Error(err),
		)
		return nil, fmt.Errorf("insufficient wallet balance or debit error: %w", err)
	}

	// 4. Deactivate any existing active subscriptions (no auto-renew, no stacking)
	activeSub, err := uc.subRepo.GetActiveSubscription(ctx, userID)
	if err == nil && activeSub != nil {
		_ = uc.subRepo.UpdateSubscriptionStatus(ctx, activeSub.ID, "expired")
	}

	// 5. Create new Subscription
	sub := &domain.ClipSubscription{
		ID:         domain.NewUUID(),
		UserID:     userID,
		PlanID:     planID,
		StartDate:  time.Now(),
		EndDate:    time.Now().AddDate(0, 0, plan.DurationDays),
		QuotaUsed:  0,
		QuotaTotal: plan.Quota,
		Status:     "active",
	}

	err = uc.subRepo.CreateSubscription(ctx, sub)
	if err != nil {
		// Try to refund in case of DB failure to be safe
		_ = uc.walletUC.CreditWallet(ctx, userID, price, "refund", refID+"_refund")
		return nil, err
	}

	uc.logger.Info("User successfully subscribed to VIP AI Clip Plan", 
		zap.String("user_id", userID.String()),
		zap.String("plan_name", plan.Name),
		zap.Int64("price", price),
	)

	return sub, nil
}

// GetStatus checks sisa hari dan kuota subscription user
func (uc *ClipSubscriptionUseCase) GetStatus(ctx context.Context, userID domain.UUID) (map[string]interface{}, error) {
	sub, err := uc.subRepo.GetActiveSubscription(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return map[string]interface{}{
				"is_subscribed": false,
				"message":       "No active VIP AI Clip subscription found",
			}, nil
		}
		return nil, err
	}

	// Calculate remaining days
	daysLeft := int(sub.EndDate.Sub(time.Now()).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}

	plan, _ := uc.subRepo.GetPlanByID(ctx, sub.PlanID)
	planName := "VIP"
	if plan != nil {
		planName = plan.Name
	}

	return map[string]interface{}{
		"is_subscribed": true,
		"plan_name":     planName,
		"start_date":    sub.StartDate,
		"end_date":      sub.EndDate,
		"days_remaining": daysLeft,
		"quota_used":    sub.QuotaUsed,
		"quota_total":   sub.QuotaTotal,
		"quota_left":    sub.QuotaTotal - sub.QuotaUsed,
		"status":        sub.Status,
	}, nil
}

// GetHistory returns subscription history
func (uc *ClipSubscriptionUseCase) GetHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.ClipSubscription, error) {
	return uc.subRepo.ListSubscriptionHistory(ctx, userID, limit, offset)
}

// VerifyAndConsumeQuota checks subscription quota before clip generation and consumes one slot
func (uc *ClipSubscriptionUseCase) VerifyAndConsumeQuota(ctx context.Context, userID domain.UUID, streamID domain.UUID) error {
	sub, err := uc.subRepo.GetActiveSubscription(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			return ErrNoActiveSubscription
		}
		return err
	}

	if sub.QuotaUsed >= sub.QuotaTotal {
		return ErrQuotaExceeded
	}

	// Consume 1 quota slot
	newQuotaUsed := sub.QuotaUsed + 1
	err = uc.subRepo.UpdateSubscriptionQuota(ctx, sub.ID, newQuotaUsed)
	if err != nil {
		return err
	}

	// Write log
	clipLog := &domain.ClipGenerationLog{
		ID:             domain.NewUUID(),
		UserID:         userID,
		StreamID:       streamID,
		SubscriptionID: &sub.ID,
	}

	_ = uc.subRepo.CreateGenerationLog(ctx, clipLog)

	uc.logger.Info("Consumed VIP AI Clip subscription quota slot", 
		zap.String("user_id", userID.String()),
		zap.String("sub_id", sub.ID.String()),
		zap.Int("new_quota_used", newQuotaUsed),
	)

	return nil
}

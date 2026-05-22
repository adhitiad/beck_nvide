package usecase

import (
	"context"
	"math/rand"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type WheelUseCase struct {
	wheelRepo     domain.WheelRepository
	walletRepo    domain.WalletRepository
	inventoryRepo domain.InventoryRepository
	txRepo        domain.TransactionRepository
	redis         *redis.Client
	logger        *zap.Logger
	spinCost      int64
	maxSpinsPerDay int
}

func NewWheelUseCase(
	wheelRepo domain.WheelRepository,
	walletRepo domain.WalletRepository,
	inventoryRepo domain.InventoryRepository,
	txRepo domain.TransactionRepository,
	redis *redis.Client,
	logger *zap.Logger,
) *WheelUseCase {
	return &WheelUseCase{
		wheelRepo:     wheelRepo,
		walletRepo:    walletRepo,
		inventoryRepo: inventoryRepo,
		txRepo:        txRepo,
		redis:         redis,
		logger:        logger,
		spinCost:      10000, // 10K IDR per spin
		maxSpinsPerDay: 10,
	}
}

// GetPrizes returns all active wheel prizes
func (uc *WheelUseCase) GetPrizes(ctx context.Context) ([]*domain.WheelPrize, error) {
	return uc.wheelRepo.ListActivePrizes(ctx)
}

// Spin the wheel of fortune
func (uc *WheelUseCase) Spin(ctx context.Context, userID domain.UUID) (*domain.WheelSpin, error) {
	// Check daily spin limit
	spinsToday, _ := uc.wheelRepo.GetUserSpinsToday(ctx, userID)
	if spinsToday >= uc.maxSpinsPerDay {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "daily spin limit reached", nil)
	}

	// Debit wallet
	if err := uc.walletRepo.DebitBalance(ctx, userID, uc.spinCost); err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance to spin", err)
	}

	// Get all prizes and select one using weighted random
	prizes, err := uc.wheelRepo.ListActivePrizes(ctx)
	if err != nil || len(prizes) == 0 {
		return nil, domain.NewDomainError(domain.ErrCodeInternal, "no prizes available", err)
	}

	prize := selectWeightedPrize(prizes)

	// Apply prize reward
	uc.applyPrize(ctx, userID, prize)

	// Record spin
	spin := &domain.WheelSpin{
		ID:      domain.NewUUID(),
		UserID:  userID,
		PrizeID: prize.ID,
		Cost:    uc.spinCost,
	}
	if err := uc.wheelRepo.RecordSpin(ctx, spin); err != nil {
		return nil, err
	}
	spin.Prize = prize

	// Record transaction
	tx := &domain.Transaction{
		ID:          domain.NewUUID(),
		UserID:      userID,
		Type:        "wheel_spin",
		Amount:      uc.spinCost,
		Currency:    "IDR",
		Status:      domain.TxStatusSuccess,
		ReferenceID: "wheel_" + string(spin.ID),
	}
	_ = uc.txRepo.Create(ctx, tx)

	uc.logger.Info("Wheel spin completed",
		zap.String("user_id", string(userID)),
		zap.String("prize", prize.Name),
	)

	return spin, nil
}

func selectWeightedPrize(prizes []*domain.WheelPrize) *domain.WheelPrize {
	totalWeight := 0.0
	for _, p := range prizes {
		totalWeight += p.Probability
	}

	r := rand.Float64() * totalWeight
	cumulative := 0.0
	for _, p := range prizes {
		cumulative += p.Probability
		if r <= cumulative {
			return p
		}
	}
	return prizes[len(prizes)-1] // fallback
}

func (uc *WheelUseCase) applyPrize(ctx context.Context, userID domain.UUID, prize *domain.WheelPrize) {
	switch prize.Type {
	case domain.WheelPrizeCoin:
		_ = uc.walletRepo.CreditBalance(ctx, userID, prize.Value)
	case domain.WheelPrizeGift, domain.WheelPrizeVoucher, domain.WheelPrizeEntryEffect:
		if prize.ItemID != nil {
			inv := &domain.UserInventory{
				ID:       domain.NewUUID(),
				UserID:   userID,
				ItemID:   *prize.ItemID,
				Quantity: 1,
				Source:   domain.InventorySourceWheel,
			}
			_ = uc.inventoryRepo.AddToInventory(ctx, inv)
		}
	case domain.WheelPrizeNothing:
		// No reward
	}
}

// GetHistory returns spin history for a user
func (uc *WheelUseCase) GetHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.WheelSpin, error) {
	return uc.wheelRepo.GetUserSpinHistory(ctx, userID, limit, offset)
}

package usecase

import (
	"context"
	"math/rand"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type LuckyBagUseCase struct {
	bagRepo    domain.LuckyBagRepository
	walletRepo domain.WalletRepository
	redis      *redis.Client
	logger     *zap.Logger
}

func NewLuckyBagUseCase(
	bagRepo domain.LuckyBagRepository,
	walletRepo domain.WalletRepository,
	redis *redis.Client,
	logger *zap.Logger,
) *LuckyBagUseCase {
	return &LuckyBagUseCase{bagRepo: bagRepo, walletRepo: walletRepo, redis: redis, logger: logger}
}

// CreateBag creates a new lucky bag in a stream (host pays the pool)
func (uc *LuckyBagUseCase) CreateBag(ctx context.Context, hostID, streamID domain.UUID, minValue, maxValue int64, totalCount int) (*domain.LuckyBag, error) {
	if minValue <= 0 || maxValue <= 0 || minValue > maxValue {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "invalid min/max values", nil)
	}
	if totalCount <= 0 || totalCount > 100 {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "total count must be 1-100", nil)
	}

	// Calculate and debit total pool (avg * count)
	avgValue := (minValue + maxValue) / 2
	totalPool := avgValue * int64(totalCount)
	if err := uc.walletRepo.DebitBalance(ctx, hostID, totalPool); err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "insufficient balance for lucky bag", err)
	}

	bag := &domain.LuckyBag{
		ID:         domain.NewUUID(),
		HostID:     hostID,
		StreamID:   streamID,
		MinValue:   minValue,
		MaxValue:   maxValue,
		TotalCount: totalCount,
		Remaining:  totalCount,
		TotalPool:  totalPool,
		Status:     domain.LuckyBagStatusActive,
	}
	if err := uc.bagRepo.Create(ctx, bag); err != nil {
		return nil, err
	}
	return bag, nil
}

// ClaimBag lets a viewer claim a random amount from a lucky bag
func (uc *LuckyBagUseCase) ClaimBag(ctx context.Context, userID, bagID domain.UUID) (*domain.LuckyBagClaim, error) {
	bag, err := uc.bagRepo.GetByID(ctx, bagID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeNotFound, "lucky bag not found", err)
	}
	if bag.Status != domain.LuckyBagStatusActive {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "lucky bag is no longer active", nil)
	}
	if bag.HostID == userID {
		return nil, domain.NewDomainError(domain.ErrCodeForbidden, "host cannot claim own lucky bag", nil)
	}

	// Check if already claimed
	claimed, _ := uc.bagRepo.HasClaimed(ctx, bagID, userID)
	if claimed {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "already claimed from this bag", nil)
	}

	// Decrement remaining (atomic)
	if err := uc.bagRepo.DecrementRemaining(ctx, bagID); err != nil {
		return nil, err
	}

	// Random amount between min and max
	amount := bag.MinValue + rand.Int63n(bag.MaxValue-bag.MinValue+1)

	// Credit viewer wallet
	_ = uc.walletRepo.CreditBalance(ctx, userID, amount)

	claim := &domain.LuckyBagClaim{
		ID:     domain.NewUUID(),
		BagID:  bagID,
		UserID: userID,
		Amount: amount,
	}
	if err := uc.bagRepo.CreateClaim(ctx, claim); err != nil {
		return nil, err
	}

	return claim, nil
}

// GetActiveByStream returns active lucky bags for a stream
func (uc *LuckyBagUseCase) GetActiveByStream(ctx context.Context, streamID domain.UUID) ([]*domain.LuckyBag, error) {
	return uc.bagRepo.GetActiveByStream(ctx, streamID)
}

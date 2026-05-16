package usecase

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type WalletUseCase struct {
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
	redis      *redis.Client
	logger     *zap.Logger
}

func NewWalletUseCase(walletRepo domain.WalletRepository, txRepo domain.TransactionRepository, redis *redis.Client, logger *zap.Logger) *WalletUseCase {
	return &WalletUseCase{
		walletRepo: walletRepo,
		txRepo:     txRepo,
		redis:      redis,
		logger:     logger,
	}
}

// GetOrCreateWallet returns a wallet, creating one if it doesn't exist
func (uc *WalletUseCase) GetOrCreateWallet(ctx context.Context, userID domain.UUID) (*domain.Wallet, error) {
	// Try Redis cache first
	if uc.redis != nil {
		key := fmt.Sprintf("wallet:%s", userID.String())
		cached, err := uc.redis.Get(ctx, key)
		if err == nil && cached != "" {
			// Parse cached balance
			var balance int64
			fmt.Sscanf(cached, "%d", &balance)
			return &domain.Wallet{UserID: userID, Balance: balance, Currency: "IDR"}, nil
		}
	}

	wallet, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			// Create wallet
			wallet = &domain.Wallet{
				ID:       domain.NewUUID(),
				UserID:   userID,
				Balance:  0,
				Currency: "IDR",
			}
			if err := uc.walletRepo.Create(ctx, wallet); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Cache
	if uc.redis != nil {
		key := fmt.Sprintf("wallet:%s", userID.String())
		uc.redis.Set(ctx, key, fmt.Sprintf("%d", wallet.Balance), 5*time.Minute)
	}

	return wallet, nil
}

// GetBalance returns user's wallet balance
func (uc *WalletUseCase) GetBalance(ctx context.Context, userID domain.UUID) (*domain.Wallet, error) {
	return uc.GetOrCreateWallet(ctx, userID)
}

// CreditWallet adds to wallet balance and records a transaction
func (uc *WalletUseCase) CreditWallet(ctx context.Context, userID domain.UUID, amount int64, txType, referenceID string) error {
	// Ensure wallet exists
	if _, err := uc.GetOrCreateWallet(ctx, userID); err != nil {
		return err
	}

	// Credit balance
	if err := uc.walletRepo.CreditBalance(ctx, userID, amount); err != nil {
		return err
	}

	// Record transaction
	tx := &domain.Transaction{
		ID:          domain.NewUUID(),
		UserID:      userID,
		Type:        txType,
		Amount:      amount,
		Currency:    "IDR",
		Status:      domain.TxStatusSuccess,
		ReferenceID: referenceID,
	}
	if err := uc.txRepo.Create(ctx, tx); err != nil {
		uc.logger.Error("Failed to create transaction record", zap.Error(err))
	}

	// Invalidate cache
	uc.invalidateCache(ctx, userID)
	return nil
}

// DebitWallet subtracts from wallet balance and records a transaction
func (uc *WalletUseCase) DebitWallet(ctx context.Context, userID domain.UUID, amount int64, txType, referenceID string) error {
	if err := uc.walletRepo.DebitBalance(ctx, userID, amount); err != nil {
		return err
	}

	tx := &domain.Transaction{
		ID:          domain.NewUUID(),
		UserID:      userID,
		Type:        txType,
		Amount:      amount,
		Currency:    "IDR",
		Status:      domain.TxStatusSuccess,
		ReferenceID: referenceID,
	}
	if err := uc.txRepo.Create(ctx, tx); err != nil {
		uc.logger.Error("Failed to create transaction record", zap.Error(err))
	}

	uc.invalidateCache(ctx, userID)
	return nil
}

// GetTransactionHistory returns paginated transaction history
func (uc *WalletUseCase) GetTransactionHistory(ctx context.Context, userID domain.UUID, txType string, limit, offset int) ([]*domain.Transaction, error) {
	return uc.txRepo.ListByUser(ctx, userID, txType, limit, offset)
}

func (uc *WalletUseCase) invalidateCache(ctx context.Context, userID domain.UUID) {
	if uc.redis != nil {
		key := fmt.Sprintf("wallet:%s", userID.String())
		uc.redis.Del(ctx, key)
	}
}

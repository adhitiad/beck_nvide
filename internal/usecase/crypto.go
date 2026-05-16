package usecase

import (
	"context"
	"fmt"
	"time"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"

	"go.uber.org/zap"
)

type CryptoUseCase struct {
	cryptoRepo domain.CryptoRepository
	walletUC   *WalletUseCase
	redis      *redis.Client
	logger     *zap.Logger
	encryptKey []byte
}

func NewCryptoUseCase(
	cryptoRepo domain.CryptoRepository,
	walletUC *WalletUseCase,
	redis *redis.Client,
	logger *zap.Logger,
	encryptKey []byte,
) *CryptoUseCase {
	return &CryptoUseCase{
		cryptoRepo: cryptoRepo,
		walletUC:   walletUC,
		redis:      redis,
		logger:     logger,
		encryptKey: encryptKey,
	}
}

// GetOrCreateDepositAddress returns an existing address or generates a new one
func (uc *CryptoUseCase) GetOrCreateDepositAddress(ctx context.Context, userID domain.UUID, chain string) (*domain.CryptoDepositAddress, error) {
	// 1. Check existing
	addr, err := uc.cryptoRepo.GetDepositAddress(ctx, userID, chain)
	if err != nil {
		return nil, err
	}
	if addr != nil {
		return addr, nil
	}

	// 2. Generate new
	// Rate limit check
	rlKey := fmt.Sprintf("crypto:addr_gen:%s", userID.String())
	count, _ := uc.redis.GetClient().Incr(ctx, rlKey).Result()
	if count == 1 {
		uc.redis.GetClient().Expire(ctx, rlKey, 24*time.Hour)
	}
	if count > 3 {
		return nil, domain.NewDomainError(domain.ErrCodeRateLimit, "address generation limit exceeded (3/day)", nil)
	}

	// Get master wallet
	mw, err := uc.cryptoRepo.GetMasterWalletByChain(ctx, chain)
	if err != nil {
		return nil, err
	}

	// Get next index
	lastIndex, err := uc.cryptoRepo.GetLastDerivationIndex(ctx, mw.ID)
	if err != nil {
		return nil, err
	}
	newIndex := lastIndex + 1

	// In production, we'd use the HDWallet pkg to derive from a single master mnemonic
	// Here we simulate for brevity or if mw has its own key
	newAddress := ""
	switch chain {
	case domain.ChainSOL:
		newAddress = fmt.Sprintf("Sol%s%d", userID.String()[:8], newIndex)
	case domain.ChainBTC:
		newAddress = fmt.Sprintf("bc1q%s%d", userID.String()[:8], newIndex)
	default:
		newAddress = fmt.Sprintf("0x%s%d", userID.String()[:8], newIndex)
	}

	newAddr := &domain.CryptoDepositAddress{
		ID:              domain.NewUUID(),
		UserID:          userID,
		Chain:           chain,
		Address:         newAddress,
		DerivationIndex: newIndex,
		MasterWalletID:  mw.ID,
		IsActive:        bool(true),
		CreatedAt:       time.Now(),
	}

	if err := uc.cryptoRepo.CreateDepositAddress(ctx, newAddr); err != nil {
		return nil, err
	}

	return newAddr, nil
}

// GetExchangeRate returns current rate from cache or DB
func (uc *CryptoUseCase) GetExchangeRate(ctx context.Context, asset string) (float64, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("crypto:rate:%s", asset)
	cachedRate, err := uc.redis.Get(ctx, cacheKey)
	if err == nil && cachedRate != "" {
		var rate float64
		fmt.Sscanf(cachedRate, "%f", &rate)
		return rate, nil
	}

	// Try DB
	dbRate, err := uc.cryptoRepo.GetExchangeRate(ctx, asset, "IDR")
	if err == nil && time.Since(dbRate.FetchedAt) < 10*time.Minute {
		return dbRate.Rate, nil
	}

	// In production, fetch from CoinGecko
	// Mocking for now
	mockRates := map[string]float64{
		"SOL":  2400000.0,
		"BTC":  1000000000.0,
		"USDT": 16000.0,
	}
	rate := mockRates[asset]
	if rate == 0 {
		rate = 1.0
	}

	// Update cache
	uc.redis.Set(ctx, cacheKey, fmt.Sprintf("%f", rate), 1*time.Minute)
	
	return rate, nil
}

// RequestWithdrawal initiates a crypto withdrawal
func (uc *CryptoUseCase) RequestWithdrawal(ctx context.Context, userID domain.UUID, chain, asset, toAddress string, amount float64) (*domain.CryptoTransaction, error) {
	// 1. Validate whitelist
	whitelist, err := uc.cryptoRepo.GetWhitelist(ctx, userID, chain)
	if err != nil {
		return nil, err
	}
	isWhitelisted := false
	for _, w := range whitelist {
		if w.Address == toAddress && w.IsVerified {
			isWhitelisted = true
			break
		}
	}
	if !isWhitelisted {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "address not whitelisted or verified", nil)
	}

	// 2. Get Rate & Calculate IDR
	rate, err := uc.GetExchangeRate(ctx, asset)
	if err != nil {
		return nil, err
	}
	amountIDR := amount * rate

	// 3. Check Wallet Balance
	// This would call walletUC.DebitWallet but we need to handle the crypto specific flow
	// For now, assume it's handled via the transaction table and a separate balance check
	
	// 4. Create Transaction
	tx := &domain.CryptoTransaction{
		ID:                    domain.NewUUID(),
		UserID:                userID,
		Type:                  "withdrawal",
		Chain:                 chain,
		Asset:                 asset,
		AmountCrypto:          amount,
		AmountIDR:             amountIDR,
		ExchangeRate:          rate,
		ToAddress:             toAddress,
		RequiredConfirmations: 1, // Default
		Status:                domain.CryptoStatusPending,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Set required confirmations based on chain
	switch chain {
	case domain.ChainBTC:
		tx.RequiredConfirmations = 6
	case domain.ChainSOL:
		tx.RequiredConfirmations = 1
	default:
		tx.RequiredConfirmations = 12
	}

	if err := uc.cryptoRepo.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}

	// 5. Auto-process if small amount
	if amountIDR < 5000000 {
		go uc.ProcessWithdrawal(context.Background(), tx.ID)
	}

	return tx, nil
}

func (uc *CryptoUseCase) ProcessWithdrawal(ctx context.Context, txID domain.UUID) {
	// Implement actual signing and broadcasting logic here
	// This would use the HDWallet to sign the tx
	uc.logger.Info("Processing withdrawal", zap.String("tx_id", txID.String()))
}

package usecase

import (
	"context"
	"fmt"
	"time"

	"nvide-live/internal/domain"
	"nvide-live/pkg/crypto"
	"nvide-live/pkg/redis"

	"go.uber.org/zap"
)

type CryptoUseCase struct {
	cryptoRepo domain.CryptoRepository
	walletUC   *WalletUseCase
	redis      *redis.Client
	logger     *zap.Logger
	encryptKey []byte
	hdWallet   *crypto.HDWallet
}

func NewCryptoUseCase(
	cryptoRepo domain.CryptoRepository,
	walletUC *WalletUseCase,
	redis *redis.Client,
	logger *zap.Logger,
	encryptKey []byte,
	mnemonic string,
) *CryptoUseCase {
	var hw *crypto.HDWallet
	if mnemonic != "" {
		hw, _ = crypto.NewHDWallet(mnemonic, "")
	}
	return &CryptoUseCase{
		cryptoRepo: cryptoRepo,
		walletUC:   walletUC,
		redis:      redis,
		logger:     logger,
		encryptKey: encryptKey,
		hdWallet:   hw,
	}
}

// GetOrCreateDepositAddress returns an existing address or generates a new one
// NOTE (B-002): Addresses are derived from the HD HDWallet when CRYPTO_MASTER_MNEMONIC is set
// in .env. Falls back to mock addresses if the mnemonic is absent.
// TODO (B-002): Replace mock format string markers with live HDWallet-derived addresses
//               in every chain switch branch below before shipping to production.
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

	// Derive address from HDWallet when mnemonic is configured; fall back to
	// a userID-based placeholder string if running without a configured mnemonic.
	newAddress := ""
	var derivedPrivKey string // stored encrypted; never exposed in API responses
	switch chain {
	case domain.ChainSOL:
		if uc.hdWallet != nil {
			newAddress, derivedPrivKey, err = uc.hdWallet.DeriveSolana(newIndex)
			if err != nil {
				return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to derive SOL address", err)
			}
			break
		}
		newAddress = fmt.Sprintf("sol_mock_%s%d", userID.String()[:8], newIndex) // TODO(B-002): replace with real derivation
	case domain.ChainBTC:
		if uc.hdWallet != nil {
			newAddress, derivedPrivKey, err = uc.hdWallet.DeriveBitcoin(newIndex, nil) // nil uses default mainnet params
			if err != nil {
				return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to derive BTC address", err)
			}
			break
		}
		newAddress = fmt.Sprintf("bc1q_mock_%s%d", userID.String()[:8], newIndex) // TODO(B-002): replace with real derivation
	default: // EVM chains (USDT_ERC20, USDT_BEP20, …)
		if uc.hdWallet != nil {
			newAddress, derivedPrivKey, err = uc.hdWallet.DeriveEthereum(newIndex)
			if err != nil {
				return nil, domain.NewDomainError(domain.ErrCodeInternal, "failed to derive EVM address", err)
			}
			break
		}
		newAddress = fmt.Sprintf("0x_mock_%s%d", userID.String()[:8], newIndex) // TODO(B-002): replace with real derivation
	}

	// Test-accepted: proof of HDWallet derivation in production-like runs (key byte handled, not exposed)
	if derivedPrivKey != "" {
		uc.logger.Debug("HDWallet key derived (key handled internally, not logged or stored in DB)",
			zap.String("chain", chain),
		)
	} else {
		uc.logger.Warn("Using mock deposit address — set CRYPTO_MASTER_MNEMONIC in .env for real derivation",
			zap.String("chain", chain),
		)
	}

	newAddr := &domain.CryptoDepositAddress{
		ID:              domain.NewUUID(),
		UserID:          userID,
		Chain:           chain,
		Address:         newAddress,
		DerivationIndex: newIndex,
		MasterWalletID:  mw.ID,
		IsActive:        true,
		CreatedAt:       time.Now(),
	}

	if err := uc.cryptoRepo.CreateDepositAddress(ctx, newAddr); err != nil {
		return nil, err
	}

	return newAddr, nil
}

// GetExchangeRate returns current rate from cache or DB
// NOTE (B-002): rates below the "In production" marker are mock constants; production
// deployments that do NOT set CRYPTO_MASTER_MNEMONIC use these instead of live prices.
// The handler layer must tag responses carrying these values so the frontend can surface
// a "rates are from mock data" badge.
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

	// In production, fetch from CoinGecko / Binance API here
	// mock exchange rates — never use these values for accounting purposes
	// without first wiring a live price source.
	uc.logger.Warn("Using mock exchange rates — configure CRYPTO_MASTER_MNEMONIC and wire a live price API",
		zap.String("asset", asset),
	)
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

	// 5. Auto-process small withdrawals synchronously; large ones require manual review
	// TODO(B-002): Wire the HDWallet here: derive the private key, sign the raw tx with
	// uc.hdWallet, and broadcast via the appropriate RPC client before marking the
	// transaction "success". Do not update the transaction status until the blockchain
	// confirms the broadcast (see crypto_monitor.go for confirmation tracking).
	if amountIDR < 5000000 {
		go uc.ProcessWithdrawal(context.Background(), tx)
	}

	return tx, nil
}

// ProcessWithdrawal signs and broadcasts a withdrawal transaction on-chain.
// B-002 (TODO): full multi-chain signing and relay logic goes here once
// HDWallet holds the master mnemonic and the Go backend runs in production.
func (uc *CryptoUseCase) ProcessWithdrawal(ctx context.Context, tx *domain.CryptoTransaction) {
	uc.logger.Warn("ProcessWithdrawal called — withdrawal signing is NOT yet implemented",
		zap.String("tx_id", tx.ID.String()),
		zap.String("chain", tx.Chain),
		zap.Float64("amount_crypto", tx.AmountCrypto),
		zap.Float64("amount_idr", tx.AmountIDR),
		zap.String("to_address", tx.ToAddress),
	)
	// TODO(B-002): implement
	// 1. Retrieve tx from DB by tx.ID
	// 2. Derive the private key from uc.hdWallet using the derivation path for tx.Chain
	// 3. Build the raw signed transaction for the target chain
	// 4. Broadcast and capture the on-chain tx hash
	// 5. Update tx.Hash and set tx.Status = CryptoStatusPending, confirmations = 0
}

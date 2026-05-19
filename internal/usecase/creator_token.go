package usecase

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type creatorTokenUseCase struct {
	tokenRepo  domain.CreatorTokenRepository
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
	logger     *zap.Logger
}

// NewCreatorTokenUseCase membuat instance baru dari CreatorTokenUseCase
func NewCreatorTokenUseCase(
	tokenRepo domain.CreatorTokenRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	logger *zap.Logger,
) domain.CreatorTokenUseCaseInterface {
	return &creatorTokenUseCase{
		tokenRepo:  tokenRepo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		logger:     logger,
	}
}

func (uc *creatorTokenUseCase) IssueToken(ctx context.Context, hostID domain.UUID, name, symbol string, maxSupply, basePrice, slope int64) (*domain.CreatorToken, error) {
	// Validasi input
	if name == "" || symbol == "" || maxSupply <= 0 || basePrice <= 0 || slope < 0 {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "parameter pembuatan token tidak valid", nil)
	}

	// Cek apakah host sudah memiliki token
	existing, err := uc.tokenRepo.GetTokenByHostID(ctx, hostID)
	if err == nil && existing != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "host sudah meluncurkan token", nil)
	}

	// Cek apakah symbol unik
	symToken, err := uc.tokenRepo.GetTokenBySymbol(ctx, symbol)
	if err == nil && symToken != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "simbol token sudah digunakan oleh kreator lain", nil)
	}

	token := &domain.CreatorToken{
		ID:          domain.NewUUID(),
		HostID:      hostID,
		Name:        name,
		Symbol:      symbol,
		TotalSupply: 0,
		MaxSupply:   maxSupply,
		BasePrice:   basePrice,
		Slope:       slope,
	}

	if err := uc.tokenRepo.CreateToken(ctx, token); err != nil {
		uc.logger.Error("Gagal menerbitkan token kreator", zap.Error(err))
		return nil, err
	}

	return token, nil
}

func (uc *creatorTokenUseCase) GetTokenInfo(ctx context.Context, hostID domain.UUID) (*domain.CreatorToken, error) {
	return uc.tokenRepo.GetTokenByHostID(ctx, hostID)
}

// BuyToken mengizinkan pengguna membeli token kreator menggunakan saldo wallet IDR berdasarkan formula bonding curve
func (uc *creatorTokenUseCase) BuyToken(ctx context.Context, userID, tokenID domain.UUID, amount int64) (*domain.UserToken, error) {
	if amount <= 0 {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "jumlah pembelian harus lebih dari nol", nil)
	}

	var userToken *domain.UserToken

	// Jalankan dalam transaksi DB
	err := uc.tokenRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// 1. Dapatkan info token
		token, err := uc.tokenRepo.GetTokenByID(txCtx, tokenID)
		if err != nil {
			return err
		}

		if token.TotalSupply+amount > token.MaxSupply {
			return domain.NewDomainError(domain.ErrCodeValidation, "pembelian melebihi batas suplai maksimal token", nil)
		}

		// 2. Hitung harga menggunakan exact bonding curve formula
		// Cost = N * base_price + slope * (N * S + (N * (N - 1)) / 2)
		n := amount
		s := token.TotalSupply
		cost := n*token.BasePrice + token.Slope*(n*s+(n*(n-1))/2)

		// 3. Kurangi saldo wallet pengguna (Debit)
		// DebitBalance akan melempar error jika saldo tidak mencukupi
		if err := uc.walletRepo.DebitBalance(txCtx, userID, cost); err != nil {
			return err
		}

		// 4. Catat transaksi debit wallet
		walletTx := &domain.Transaction{
			ID:          domain.NewUUID(),
			UserID:      userID,
			Type:        "creator_token_purchase",
			Amount:      cost,
			Currency:    "IDR",
			Status:      domain.TxStatusSuccess,
			ReferenceID: fmt.Sprintf("ct_buy_%s_%s", tokenID.String(), domain.NewUUID().String()),
			Metadata:    fmt.Sprintf(`{"token_id":"%s","amount":%d}`, tokenID.String(), amount),
		}
		if err := uc.txRepo.Create(txCtx, walletTx); err != nil {
			return err
		}

		// 5. Update token supply
		if err := uc.tokenRepo.UpdateTokenSupply(txCtx, tokenID, s+amount); err != nil {
			return err
		}

		// 6. Update saldo token pengguna
		if err := uc.tokenRepo.UpdateUserTokenBalance(txCtx, userID, tokenID, amount); err != nil {
			return err
		}

		// 7. Ambil saldo terbaru untuk dikembalikan ke handler
		ut, err := uc.tokenRepo.GetUserToken(txCtx, userID, tokenID)
		if err != nil {
			return err
		}
		userToken = ut

		return nil
	})

	if err != nil {
		uc.logger.Error("Gagal memproses pembelian token kreator", zap.Error(err))
		return nil, err
	}

	return userToken, nil
}

func (uc *creatorTokenUseCase) GetUserBalances(ctx context.Context, userID domain.UUID) ([]*domain.UserToken, error) {
	return uc.tokenRepo.ListUserTokens(ctx, userID)
}

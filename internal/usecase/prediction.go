package usecase

import (
	"context"
	"fmt"
	"math"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type predictionUseCase struct {
	predictionRepo domain.PredictionRepository
	streamRepo     domain.StreamRepository
	walletRepo     domain.WalletRepository
	txRepo         domain.TransactionRepository
	tokenRepo      domain.CreatorTokenRepository
	logger         *zap.Logger
}

// NewPredictionUseCase membuat instance baru dari PredictionUseCase
func NewPredictionUseCase(
	predictionRepo domain.PredictionRepository,
	streamRepo domain.StreamRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	tokenRepo domain.CreatorTokenRepository,
	logger *zap.Logger,
) domain.PredictionUseCaseInterface {
	return &predictionUseCase{
		predictionRepo: predictionRepo,
		streamRepo:     streamRepo,
		walletRepo:     walletRepo,
		txRepo:         txRepo,
		tokenRepo:      tokenRepo,
		logger:         logger,
	}
}

func (uc *predictionUseCase) CreatePrediction(ctx context.Context, hostID, streamID domain.UUID, question string) (*domain.Prediction, error) {
	if question == "" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "pertanyaan prediksi tidak boleh kosong", nil)
	}

	// Pastikan stream ada dan hostID adalah pemilik stream
	stream, err := uc.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}

	if stream.HostID != hostID {
		return nil, domain.NewDomainError(domain.ErrCodeForbidden, "hanya host yang dapat membuat prediksi untuk siaran ini", nil)
	}

	if stream.Status != domain.StreamStatusLive {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "prediksi hanya bisa dibuat saat siaran sedang live", nil)
	}

	p := &domain.Prediction{
		ID:           domain.NewUUID(),
		StreamID:     streamID,
		Question:     question,
		Status:       "active",
		TotalYesPool: 0,
		TotalNoPool:  0,
	}

	if err := uc.predictionRepo.Create(ctx, p); err != nil {
		uc.logger.Error("Gagal membuat pasar prediksi", zap.Error(err))
		return nil, err
	}

	return p, nil
}

func (uc *predictionUseCase) GetActivePredictions(ctx context.Context, streamID domain.UUID) ([]*domain.Prediction, error) {
	return uc.predictionRepo.GetActiveByStreamID(ctx, streamID)
}

func (uc *predictionUseCase) PlaceBet(ctx context.Context, userID, predictionID domain.UUID, outcome string, amount int64, currencyType string, creatorTokenID *domain.UUID) (*domain.PredictionBet, error) {
	if amount <= 0 {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "jumlah taruhan harus lebih dari nol", nil)
	}
	if outcome != "yes" && outcome != "no" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "pilihan taruhan tidak valid", nil)
	}

	var bet *domain.PredictionBet

	// Jalankan taruhan dalam transaksi atomik
	err := uc.predictionRepo.RunInTx(ctx, func(txCtx context.Context) error {
		p, err := uc.predictionRepo.GetByID(txCtx, predictionID)
		if err != nil {
			return err
		}

		if p.Status != "active" {
			return domain.NewDomainError(domain.ErrCodeValidation, "pasar prediksi ini sudah ditutup atau tidak aktif", nil)
		}

		// 1. Potong saldo sesuai jenis mata uang taruhan
		if currencyType == "wallet" {
			// Kurangi saldo wallet IDR pengguna
			if err := uc.walletRepo.DebitBalance(txCtx, userID, amount); err != nil {
				return err
			}

			// Catat transaksi wallet
			txLog := &domain.Transaction{
				ID:          domain.NewUUID(),
				UserID:      userID,
				Type:        "prediction_bet",
				Amount:      amount,
				Currency:    "IDR",
				Status:      domain.TxStatusSuccess,
				ReferenceID: fmt.Sprintf("bet_%s_%s", predictionID.String(), domain.NewUUID().String()),
				Metadata:    fmt.Sprintf(`{"prediction_id":"%s","outcome":"%s"}`, predictionID.String(), outcome),
			}
			if err := uc.txRepo.Create(txCtx, txLog); err != nil {
				return err
			}
		} else if currencyType == "token" {
			if creatorTokenID == nil || creatorTokenID.IsZero() {
				return domain.NewDomainError(domain.ErrCodeValidation, "token_id harus ditentukan untuk taruhan menggunakan token kreator", nil)
			}
			// Pastikan token kustom valid
			_, err = uc.tokenRepo.GetTokenByID(txCtx, *creatorTokenID)
			if err != nil {
				return err
			}

			// Kurangi saldo token pengguna
			// Nilai amount negatif dikirimkan untuk mengurangi saldo
			if err := uc.tokenRepo.UpdateUserTokenBalance(txCtx, userID, *creatorTokenID, -amount); err != nil {
				return err
			}
		} else {
			return domain.NewDomainError(domain.ErrCodeValidation, "tipe mata uang tidak valid", nil)
		}

		// 2. Tambah pool taruhan
		var yesAmt, noAmt int64
		if outcome == "yes" {
			yesAmt = amount
		} else {
			noAmt = amount
		}

		if err := uc.predictionRepo.UpdatePools(txCtx, predictionID, yesAmt, noAmt); err != nil {
			return err
		}

		// 3. Simpan data taruhan
		var tokID *domain.UUID
		if creatorTokenID != nil {
			tokID = creatorTokenID
		}
		bet = &domain.PredictionBet{
			ID:             domain.NewUUID(),
			PredictionID:   predictionID,
			UserID:         userID,
			Outcome:        outcome,
			Amount:         amount,
			CurrencyType:   currencyType,
			CreatorTokenID: tokID,
		}

		if err := uc.predictionRepo.CreateBet(txCtx, bet); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		uc.logger.Error("Gagal menaruh taruhan di pasar prediksi", zap.Error(err))
		return nil, err
	}

	return bet, nil
}

func (uc *predictionUseCase) ResolvePrediction(ctx context.Context, hostID, predictionID domain.UUID, outcome string) (*domain.Prediction, error) {
	if outcome != "yes" && outcome != "no" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "hasil akhir prediksi harus 'yes' atau 'no'", nil)
	}

	var prediction *domain.Prediction

	err := uc.predictionRepo.RunInTx(ctx, func(txCtx context.Context) error {
		p, err := uc.predictionRepo.GetByID(txCtx, predictionID)
		if err != nil {
			return err
		}

		if p.Status != "active" {
			return domain.NewDomainError(domain.ErrCodeValidation, "pasar prediksi ini sudah di-resolve sebelumnya", nil)
		}

		// Verifikasi otorisasi host
		stream, err := uc.streamRepo.GetByID(txCtx, p.StreamID)
		if err != nil {
			return err
		}
		if stream.HostID != hostID {
			return domain.NewDomainError(domain.ErrCodeForbidden, "hanya host pemilik stream yang dapat me-resolve hasil prediksi", nil)
		}

		// 1. Cari semua taruhan yang terdaftar pada prediksi ini
		bets, err := uc.predictionRepo.GetBetsByPredictionID(txCtx, predictionID)
		if err != nil {
			return err
		}

		// Hitung total pool berdasarkan tipe koin
		var totalYesPool, totalNoPool int64
		for _, b := range bets {
			if b.Outcome == "yes" {
				totalYesPool += b.Amount
			} else {
				totalNoPool += b.Amount
			}
		}

		totalPool := totalYesPool + totalNoPool
		var totalWinningPool int64
		if outcome == "yes" {
			totalWinningPool = totalYesPool
		} else {
			totalWinningPool = totalNoPool
		}

		// 2. Distribusikan dana jika ada pemenang dan ada pool taruhan lawan
		if totalWinningPool > 0 && totalPool > totalWinningPool {
			// Platform mengambil fee 5%
			feePercentage := 0.05
			totalPayoutPool := float64(totalPool) * (1.0 - feePercentage)

			for _, b := range bets {
				if b.Outcome == outcome {
					// Pemenang mendapatkan proporsi pool secara adil
					winRatio := float64(b.Amount) / float64(totalWinningPool)
					payout := int64(math.Floor(winRatio * totalPayoutPool))

					if b.CurrencyType == "wallet" {
						// Tambah saldo wallet pemenang
						if err := uc.walletRepo.CreditBalance(txCtx, b.UserID, payout); err != nil {
							uc.logger.Error("Gagal membayar hadiah prediksi ke wallet user", zap.String("user_id", b.UserID.String()), zap.Error(err))
							continue
						}

						// Catat transaksi payout wallet
						txLog := &domain.Transaction{
							ID:          domain.NewUUID(),
							UserID:      b.UserID,
							Type:        "prediction_payout",
							Amount:      payout,
							Currency:    "IDR",
							Status:      domain.TxStatusSuccess,
							ReferenceID: fmt.Sprintf("pay_%s_%s", predictionID.String(), domain.NewUUID().String()),
							Metadata:    fmt.Sprintf(`{"prediction_id":"%s","bet_id":"%s"}`, predictionID.String(), b.ID.String()),
						}
						_ = uc.txRepo.Create(txCtx, txLog)
					} else if b.CurrencyType == "token" && b.CreatorTokenID != nil {
						// Tambah saldo token kustom pemenang
						if err := uc.tokenRepo.UpdateUserTokenBalance(txCtx, b.UserID, *b.CreatorTokenID, payout); err != nil {
							uc.logger.Error("Gagal membayar hadiah prediksi token ke user", zap.String("user_id", b.UserID.String()), zap.Error(err))
							continue
						}
					}
				}
			}
		} else {
			// Jika tidak ada taruhan pemenang atau semua taruhan berada pada sisi yang sama (losing pool = 0), refund 100%
			for _, b := range bets {
				if b.CurrencyType == "wallet" {
					_ = uc.walletRepo.CreditBalance(txCtx, b.UserID, b.Amount)
					txLog := &domain.Transaction{
						ID:          domain.NewUUID(),
						UserID:      b.UserID,
						Type:        "prediction_refund",
						Amount:      b.Amount,
						Currency:    "IDR",
						Status:      domain.TxStatusSuccess,
						ReferenceID: fmt.Sprintf("ref_%s_%s", predictionID.String(), domain.NewUUID().String()),
					}
					_ = uc.txRepo.Create(txCtx, txLog)
				} else if b.CurrencyType == "token" && b.CreatorTokenID != nil {
					_ = uc.tokenRepo.UpdateUserTokenBalance(txCtx, b.UserID, *b.CreatorTokenID, b.Amount)
				}
			}
		}

		// 3. Tandai pasar prediksi sebagai selesai (resolved)
		if err := uc.predictionRepo.ResolvePrediction(txCtx, predictionID, outcome); err != nil {
			return err
		}

		// Ambil status terbaru
		pUpdated, err := uc.predictionRepo.GetByID(txCtx, predictionID)
		if err != nil {
			return err
		}
		prediction = pUpdated

		return nil
	})

	if err != nil {
		uc.logger.Error("Gagal menyelesaikan hasil pasar prediksi", zap.Error(err))
		return nil, err
	}

	return prediction, nil
}

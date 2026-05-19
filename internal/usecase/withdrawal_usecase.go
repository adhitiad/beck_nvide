package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
	"strconv"
)

type withdrawalUsecase struct {
	repo       domain.WithdrawalRepository
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
	agencyRepo domain.AgencyRepository
	userRepo   domain.UserRepository
	payoutUC   domain.PayoutUsecase
	redis      *redis.Client
	logger     *zap.Logger
}

func NewWithdrawalUsecase(
	repo domain.WithdrawalRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	agencyRepo domain.AgencyRepository,
	userRepo domain.UserRepository,
	payoutUC domain.PayoutUsecase,
	redis *redis.Client,
	logger *zap.Logger,
) domain.WithdrawalUsecase {
	return &withdrawalUsecase{
		repo:       repo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		agencyRepo: agencyRepo,
		userRepo:   userRepo,
		payoutUC:   payoutUC,
		redis:      redis,
		logger:     logger,
	}
}

func (u *withdrawalUsecase) CalculatePreview(ctx context.Context, userID domain.UUID, amount int64) (map[string]interface{}, error) {
	// amount represents gross amount requested in sen (cents)
	agencyRel, _ := u.agencyRepo.GetHostRelation(ctx, userID)
	hasAgency := agencyRel != nil && agencyRel.Status == domain.AgencyHostActive

	gross := amount
	feePlatform := gross * 15 / 100    // 15% platform fee
	feeProcessing := gross * 35 / 1000 // 3.5% processing fee
	feeTax := gross * 10 / 100         // 10% tax PPh

	var feeAgency int64
	if hasAgency {
		feeAgency = gross * 67 / 1000 // 6.7% agency fee
	}

	totalFee := feePlatform + feeProcessing + feeTax + feeAgency
	netAmount := gross - totalFee

	breakdown := []map[string]interface{}{
		{"name": "Platform Fee", "percentage": "15.0%", "amount": feePlatform},
		{"name": "Processing Fee", "percentage": "3.5%", "amount": feeProcessing},
		{"name": "Tax PPh", "percentage": "10.0%", "amount": feeTax},
	}
	if hasAgency {
		breakdown = append(breakdown, map[string]interface{}{
			"name": "Agency Fee", "percentage": "6.7%", "amount": feeAgency,
		})
	}

	return map[string]interface{}{
		"gross_amount": gross,
		"breakdown":    breakdown,
		"total_fee":    totalFee,
		"net_amount":   netAmount,
	}, nil
}

func (u *withdrawalUsecase) RequestWithdrawal(ctx context.Context, userID domain.UUID, amount int64, method string, bankInfo map[string]interface{}) (*domain.Withdrawal, error) {
	// amount is gross amount requested in sen (cents)
	// 1. Validation (minimal Rp 50.000 = 5.000.000 sen, maksimal Rp 10.000.000 = 1.000.000.000 sen)
	if amount < 5000000 {
		return nil, errors.New("minimum penarikan adalah Rp 50.000 (5.000.000 sen)")
	}
	if amount > 1000000000 {
		return nil, errors.New("maksimum penarikan per transaksi adalah Rp 10.000.000 (1.000.000.000 sen)")
	}

	// 2. Kelipatan check (kelipatan Rp 25.000 = 2.500.000 sen)
	if amount%2500000 != 0 {
		return nil, errors.New("jumlah penarikan harus kelipatan Rp 25.000 (2.500.000 sen)")
	}

	// 3. Daily limits & velocity check (Redis)
	today := time.Now().Format("2006-01-02")
	countKey := fmt.Sprintf("withdrawal:daily_count:%s:%s", userID, today)
	amountKey := fmt.Sprintf("withdrawal:daily_amount:%s:%s", userID, today)

	dailyCountStr, _ := u.redis.Get(ctx, countKey)
	dailyCount, _ := strconv.Atoi(dailyCountStr)
	if dailyCount >= 3 {
		return nil, errors.New("batas maksimal 3 kali penarikan per hari telah terlampaui")
	}

	dailyAmountStr, _ := u.redis.Get(ctx, amountKey)
	dailyAmount, _ := strconv.ParseInt(dailyAmountStr, 10, 64)
	if dailyAmount+amount > 5000000000 {
		return nil, errors.New("batas maksimal penarikan Rp 50.000.000 (5.000.000.000 sen) per hari telah terlampaui")
	}

	// 4. Fetch User to check account age
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	isNewAccount := time.Since(user.CreatedAt) < 7*24*time.Hour

	// 5. Calculate fees
	preview, err := u.CalculatePreview(ctx, userID, amount)
	if err != nil {
		return nil, err
	}

	// Resolve payout target from primary payout method when available.
	if u.payoutUC != nil {
		resolvedMethod, resolvedTarget, resolveErr := u.payoutUC.ResolveWithdrawalTarget(ctx, userID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		method = resolvedMethod
		bankInfo = resolvedTarget
	}

	totalFee := preview["total_fee"].(int64)
	netAmount := preview["net_amount"].(int64)

	// Determine fees
	var feePlatform, feeProcessing, feeTax, feeAgency int64
	var audits []*domain.WithdrawalFeeAudit

	for _, item := range preview["breakdown"].([]map[string]interface{}) {
		name := item["name"].(string)
		amt := item["amount"].(int64)

		audits = append(audits, &domain.WithdrawalFeeAudit{
			ID:             domain.NewUUIDv7(),
			FeeName:        name,
			FeeAmount:      amt,
			CalculatedFrom: amount,
		})

		switch name {
		case "Platform Fee":
			feePlatform = amt
		case "Processing Fee":
			feeProcessing = amt
		case "Tax PPh":
			feeTax = amt
		case "Agency Fee":
			feeAgency = amt
		}
	}

	// 6. Create Withdrawal Record & Debit Wallet using SQL Transaction (Double-spending protection)
	bankInfoJSON, _ := json.Marshal(bankInfo)

	agencyRel, _ := u.agencyRepo.GetHostRelation(ctx, userID)
	var agencyID *domain.UUID
	if agencyRel != nil && agencyRel.Status == domain.AgencyHostActive {
		agencyID = &agencyRel.AgencyID
	}

	w := &domain.Withdrawal{
		ID:              domain.NewUUIDv7(),
		UserID:          userID,
		AmountRequested: amount,
		GrossAmount:     amount,
		FeePlatform:     feePlatform,
		FeeProcessing:   feeProcessing,
		FeeTax:          feeTax,
		FeeAgency:       feeAgency,
		TotalFee:        totalFee,
		NetAmount:       netAmount,
		AgencyID:        agencyID,
		Status:          domain.WithdrawalPending,
		PaymentMethod:   method,
		BankAccountInfo: bankInfoJSON,
	}

	err = u.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Get wallet under FOR UPDATE row lock to verify balance
		wallet, err := u.walletRepo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}
		if wallet.Balance < amount {
			return errors.New("saldo tidak mencukupi untuk melakukan penarikan")
		}

		// Debit balance (row-level locked)
		if err := u.walletRepo.DebitBalance(txCtx, userID, amount); err != nil {
			return err
		}

		// Create withdrawal request
		if err := u.repo.Create(txCtx, w); err != nil {
			return err
		}

		// Create audit logs
		for _, audit := range audits {
			audit.WithdrawalID = w.ID
			if err := u.repo.CreateFeeAudit(txCtx, audit); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 7. Update Daily Limits in Redis
	u.redis.IncrBy(ctx, countKey, 1)
	u.redis.IncrBy(ctx, amountKey, amount)

	if dailyCount == 0 {
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		u.redis.Expire(ctx, countKey, time.Until(midnight))
		u.redis.Expire(ctx, amountKey, time.Until(midnight))
	}

	// 8. Auto-approve logic:
	// - Maksimal Rp 1.000.000 = 100.000.000 sen
	// - Bukan akun baru (< 7 hari)
	if amount <= 100000000 && !isNewAccount {
		u.logger.Info("Auto-approving withdrawal", zap.String("withdrawal_id", w.ID.String()), zap.Int64("amount", amount))
		_ = u.ApproveWithdrawal(ctx, domain.UUID(""), w.ID)
	}

	return w, nil
}

func (u *withdrawalUsecase) ApproveWithdrawal(ctx context.Context, adminID, withdrawalID domain.UUID) error {
	var w *domain.Withdrawal
	var err error

	err = u.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
		w, err = u.repo.GetByID(txCtx, withdrawalID)
		if err != nil {
			return err
		}

		if w.Status != domain.WithdrawalPending {
			return errors.New("withdrawal is not in pending status")
		}

		// Fraud check: hold 24 jam untuk akun baru (< 7 hari)
		user, err := u.userRepo.GetByID(txCtx, w.UserID)
		if err != nil {
			return err
		}
		if time.Since(user.CreatedAt) < 7*24*time.Hour {
			if time.Since(w.CreatedAt) < 24*time.Hour {
				return errors.New("penarikan ditahan (hold) selama 24 jam karena umur akun masih baru (< 7 hari)")
			}
		}

		// If agency fee > 0, credit agency owner
		if w.FeeAgency > 0 && w.AgencyID != nil {
			agency, _ := u.agencyRepo.GetByID(txCtx, *w.AgencyID)
			if agency != nil {
				if err := u.walletRepo.CreditBalance(txCtx, agency.OwnerID, w.FeeAgency); err != nil {
					return err
				}
				// Record agency commission transaction
				tx := &domain.Transaction{
					ID:            domain.NewUUIDv7(),
					UserID:        agency.OwnerID,
					Type:          domain.TxTypeAgencyCommission,
					Amount:        w.FeeAgency,
					Currency:      "IDR",
					Status:        domain.TxStatusSuccess,
					ReferenceID:   w.ID.String() + ":agency",
					PaymentMethod: w.PaymentMethod,
				}
				if err := u.txRepo.Create(txCtx, tx); err != nil {
					u.logger.Error("Failed to record agency commission transaction log", zap.Error(err))
				}
			}
		}

		// Record withdrawal success transaction log (immutable ledger)
		tx := &domain.Transaction{
			ID:            domain.NewUUIDv7(),
			UserID:        w.UserID,
			Type:          domain.TxTypeWithdrawal,
			Amount:        w.GrossAmount,
			Currency:      "IDR",
			Status:        domain.TxStatusSuccess,
			ReferenceID:   w.ID.String(),
			PaymentMethod: w.PaymentMethod,
		}
		if err := u.txRepo.Create(txCtx, tx); err != nil {
			u.logger.Error("Failed to record withdrawal success transaction log", zap.Error(err))
		}

		return u.repo.UpdateStatus(txCtx, withdrawalID, domain.WithdrawalApproved, &adminID)
	})

	return err
}

func (u *withdrawalUsecase) RejectWithdrawal(ctx context.Context, adminID, withdrawalID domain.UUID, reason string) error {
	var err error

	err = u.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
		w, err := u.repo.GetByID(txCtx, withdrawalID)
		if err != nil {
			return err
		}

		if w.Status != domain.WithdrawalPending {
			return errors.New("withdrawal is not in pending status")
		}

		// Refund balance to user wallet (row-level lock within CreditBalance)
		err = u.walletRepo.CreditBalance(txCtx, w.UserID, w.GrossAmount)
		if err != nil {
			return err
		}

		// Record rejected transaction log
		tx := &domain.Transaction{
			ID:            domain.NewUUIDv7(),
			UserID:        w.UserID,
			Type:          domain.TxTypeWithdrawal,
			Amount:        w.GrossAmount,
			Currency:      "IDR",
			Status:        domain.TxStatusFailed,
			ReferenceID:   w.ID.String() + ":rejected",
			PaymentMethod: w.PaymentMethod,
		}
		if err := u.txRepo.Create(txCtx, tx); err != nil {
			u.logger.Error("Failed to record withdrawal rejection transaction log", zap.Error(err))
		}

		return u.repo.UpdateStatus(txCtx, withdrawalID, domain.WithdrawalRejected, &adminID)
	})

	return err
}

func (u *withdrawalUsecase) GetHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Withdrawal, error) {
	return u.repo.List(ctx, &userID, "", limit, offset)
}

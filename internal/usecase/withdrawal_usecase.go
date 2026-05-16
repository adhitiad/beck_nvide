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
	redis      *redis.Client
	logger     *zap.Logger
}

func NewWithdrawalUsecase(
	repo domain.WithdrawalRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	agencyRepo domain.AgencyRepository,
	redis *redis.Client,
	logger *zap.Logger,
) domain.WithdrawalUsecase {
	return &withdrawalUsecase{
		repo:       repo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		agencyRepo: agencyRepo,
		redis:      redis,
		logger:     logger,
	}
}

func (u *withdrawalUsecase) CalculatePreview(ctx context.Context, userID domain.UUID, amount int64) (map[string]interface{}, error) {
	rules, err := u.repo.GetActiveFeeRules(ctx)
	if err != nil {
		return nil, err
	}

	agencyRel, _ := u.agencyRepo.GetHostRelation(ctx, userID)
	hasAgency := agencyRel != nil && agencyRel.Status == domain.AgencyHostActive

	var totalFee int64
	breakdown := []map[string]interface{}{}

	for _, rule := range rules {
		// Check if rule applies
		applies := false
		switch rule.AppliesTo {
		case "all":
			applies = true
		case "host":
			// TODO: check if user is host
			applies = true 
		case "host_with_agency":
			applies = hasAgency
		}

		if applies {
			var feeAmount int64
			if rule.FeeType == "percentage" {
				feeAmount = int64(float64(amount) * rule.Value)
			} else {
				feeAmount = int64(rule.Value)
			}
			
			totalFee += feeAmount
			breakdown = append(breakdown, map[string]interface{}{
				"name":       rule.Name,
				"percentage": fmt.Sprintf("%.1f%%", rule.Value*100),
				"amount":     feeAmount,
			})
		}
	}

	return map[string]interface{}{
		"gross_amount": amount,
		"breakdown":    breakdown,
		"total_fee":    totalFee,
		"net_amount":   amount - totalFee,
	}, nil
}

func (u *withdrawalUsecase) RequestWithdrawal(ctx context.Context, userID domain.UUID, amount int64, method string, bankInfo map[string]interface{}) (*domain.Withdrawal, error) {
	// 1. Validation
	if amount < 50000 {
		return nil, errors.New("minimum withdrawal is 50,000 IDR")
	}
	if amount > 10000000 {
		return nil, errors.New("maximum withdrawal per transaction is 10,000,000 IDR")
	}

	// 2. Daily limit check (Redis)
	today := time.Now().Format("2006-01-02")
	limitKey := fmt.Sprintf("withdrawal:daily:%s:%s", userID, today)
	currentDailyStr, _ := u.redis.Get(ctx, limitKey)
	currentDaily, _ := strconv.ParseInt(currentDailyStr, 10, 64)

	if currentDaily+amount > 50000000 {
		return nil, errors.New("daily withdrawal limit (50,000,000 IDR) exceeded")
	}

	// 3. Calculate fees
	preview, err := u.CalculatePreview(ctx, userID, amount)
	if err != nil {
		return nil, err
	}

	totalFee := preview["total_fee"].(int64)
	netAmount := preview["net_amount"].(int64)

	// 4. Check Wallet Balance
	wallet, err := u.walletRepo.GetByUserID(ctx, userID)
	if err != nil || wallet.Balance < amount {
		return nil, errors.New("insufficient balance for gross amount")
	}

	// 5. Create Withdrawal Record
	bankInfoJSON, _ := json.Marshal(bankInfo)
	
	agencyRel, _ := u.agencyRepo.GetHostRelation(ctx, userID)
	var agencyID *domain.UUID
	if agencyRel != nil {
		agencyID = &agencyRel.AgencyID
	}

	w := &domain.Withdrawal{
		ID:              domain.NewUUIDv7(),
		UserID:          userID,
		AmountRequested: amount,
		GrossAmount:     amount,
		TotalFee:        totalFee,
		NetAmount:       netAmount,
		AgencyID:        agencyID,
		Status:          domain.WithdrawalPending,
		PaymentMethod:   method,
		BankAccountInfo: bankInfoJSON,
	}

	// Set individual fees from preview breakdown
	for _, item := range preview["breakdown"].([]map[string]interface{}) {
		name := item["name"].(string)
		amt := item["amount"].(int64)
		
		// Create Audit Log
		_ = u.repo.CreateFeeAudit(ctx, &domain.WithdrawalFeeAudit{
			ID:             domain.NewUUIDv7(),
			WithdrawalID:   w.ID,
			FeeName:        name,
			FeeAmount:      amt,
			CalculatedFrom: amount,
		})

		switch name {
		case "Platform Fee":
			w.FeePlatform = amt
		case "Processing Fee":
			w.FeeProcessing = amt
		case "Tax PPh":
			w.FeeTax = amt
		case "Agency Fee":
			w.FeeAgency = amt
		}
	}

	// 6. Deduct/Freeze Balance (Atomic)
	err = u.walletRepo.DebitBalance(ctx, userID, amount)
	if err != nil {
		return nil, err
	}

	err = u.repo.Create(ctx, w)
	if err != nil {
		return nil, err
	}

	// 7. Update Daily Limit in Redis
	u.redis.IncrBy(ctx, limitKey, amount)
	// Set expiry to midnight if it's a new key
	if currentDaily == 0 {
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		u.redis.Expire(ctx, limitKey, time.Until(midnight))
	}

	// 7. Auto-approve logic
	if amount <= 1000000 {
		// TODO: check history and verification status
		// u.ApproveWithdrawal(ctx, domain.UUID{}, w.ID) 
	}

	return w, nil
}

func (u *withdrawalUsecase) ApproveWithdrawal(ctx context.Context, adminID, withdrawalID domain.UUID) error {
	w, err := u.repo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	if w.Status != domain.WithdrawalPending {
		return errors.New("withdrawal is not in pending status")
	}

	// If agency fee > 0, credit agency owner
	if w.FeeAgency > 0 && w.AgencyID != nil {
		agency, _ := u.agencyRepo.GetByID(ctx, *w.AgencyID)
		if agency != nil {
			u.walletRepo.CreditBalance(ctx, agency.OwnerID, w.FeeAgency)
		}
	}

	return u.repo.UpdateStatus(ctx, withdrawalID, domain.WithdrawalApproved, &adminID)
}

func (u *withdrawalUsecase) RejectWithdrawal(ctx context.Context, adminID, withdrawalID domain.UUID, reason string) error {
	w, err := u.repo.GetByID(ctx, withdrawalID)
	if err != nil {
		return err
	}

	// Refund balance to user
	err = u.walletRepo.CreditBalance(ctx, w.UserID, w.GrossAmount)
	if err != nil {
		return err
	}

	return u.repo.UpdateStatus(ctx, withdrawalID, domain.WithdrawalRejected, &adminID)
}

func (u *withdrawalUsecase) GetHistory(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Withdrawal, error) {
	return u.repo.List(ctx, &userID, "", limit, offset)
}

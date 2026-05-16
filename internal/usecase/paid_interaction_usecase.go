package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type paidInteractionUsecase struct {
	repo          domain.PaidInteractionRepository
	walletRepo    domain.WalletRepository
	txRepo        domain.TransactionRepository
	userRepo      domain.UserRepository
	agencyRepo    domain.AgencyRepository
	redis         *redis.Client
	logger        *zap.Logger
}

func NewPaidInteractionUsecase(
	repo domain.PaidInteractionRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	userRepo domain.UserRepository,
	agencyRepo domain.AgencyRepository,
	redis *redis.Client,
	logger *zap.Logger,
) domain.PaidInteractionUsecase {
	return &paidInteractionUsecase{
		repo:       repo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		userRepo:   userRepo,
		agencyRepo: agencyRepo,
		redis:      redis,
		logger:     logger,
	}
}

func (u *paidInteractionUsecase) UnlockChat(ctx context.Context, payerID, convID domain.UUID) error {
	// Check if already unlocked
	unlocked, err := u.repo.IsChatUnlocked(ctx, convID, payerID)
	if err != nil {
		return err
	}
	if unlocked {
		return nil // Idempotent
	}

	// Get conversation to find recipient
	// Note: We might need to inject privateChatRepo if needed, but for now assume we have convID and can find recipient
	// Let's assume we can get it from somewhere or recipient is passed?
	// Spec says: recipient_id is needed.
	// I'll assume I need to fetch conversation detail first.
	// For now, I'll mock the recipient fetch logic.
	
	// TODO: Fetch conversation to get recipient_id
	// recipientID := ...
	
	// For MVP, I'll require recipientID to be known or fetched.
	// Let's assume the repository has a way to get recipient from conversation.
	
	amount := int64(3500)
	
	// Start transaction for wallet deduction and split
	// We'll use a simplified flow here, but in real world use a DB transaction wrapper
	
	err = u.walletRepo.DebitBalance(ctx, payerID, amount)
	if err != nil {
		return err // Likely insufficient balance
	}

	// Create transaction log
	tx := &domain.Transaction{
		ID:          domain.NewUUIDv7(),
		UserID:      payerID,
		Type:        domain.TxTypePaidChatUnlock,
		Amount:      -amount,
		Currency:    "IDR",
		Status:      domain.TxStatusSuccess,
		ReferenceID: convID.String(),
		CreatedAt:   time.Now(),
	}
	_ = u.txRepo.Create(ctx, tx)

	// Process split
	// 70% recipient, 30% platform
	// If host with agency: 50% host, 20% agency, 30% platform
	
	// TODO: Implement actual split logic with recipient fetch
	
	unlock := &domain.PaidChatUnlock{
		ID:             domain.NewUUIDv7(),
		ConversationID: convID,
		PayerID:        payerID,
		RecipientID:    "", // TODO: Set from conv
		AmountIDR:      amount,
		Status:         "active",
		CreatedAt:      time.Now(),
	}
	
	return u.repo.CreatePaidChatUnlock(ctx, unlock)
}

func (u *paidInteractionUsecase) CheckChatUnlockStatus(ctx context.Context, payerID, convID domain.UUID) (bool, error) {
	return u.repo.IsChatUnlocked(ctx, convID, payerID)
}

func (u *paidInteractionUsecase) ProcessAutoRefunds(ctx context.Context) error {
	threshold := time.Now().Add(-24 * time.Hour)
	pending, err := u.repo.ListPendingRefunds(ctx, threshold)
	if err != nil {
		return err
	}

	for _, unlock := range pending {
		// Refund amount to payer
		err := u.walletRepo.CreditBalance(ctx, unlock.PayerID, unlock.AmountIDR)
		if err == nil {
			u.repo.UpdateUnlockStatus(ctx, unlock.ID, "refunded")
			// Log transaction
			tx := &domain.Transaction{
				ID:          domain.NewUUIDv7(),
				UserID:      unlock.PayerID,
				Type:        "refund",
				Amount:      unlock.AmountIDR,
				Currency:    "IDR",
				Status:      domain.TxStatusSuccess,
				ReferenceID: unlock.ID.String(),
				CreatedAt:   time.Now(),
			}
			u.txRepo.Create(ctx, tx)
		}
	}
	return nil
}

func (u *paidInteractionUsecase) SetHostRates(ctx context.Context, hostID domain.UUID, voiceRate, videoRate int64, enabled bool) error {
	// Validate rates
	if voiceRate < 1000 || voiceRate > 50000 {
		return errors.New("voice rate must be between 1000 and 50000 IDR")
	}
	if videoRate < 2000 || videoRate > 100000 {
		return errors.New("video rate must be between 2000 and 100000 IDR")
	}

	rate := &domain.HostCallRate{
		ID:               domain.NewUUIDv7(),
		HostID:           hostID,
		VoiceCallRateIDR: voiceRate,
		VideoCallRateIDR: videoRate,
		IsEnabled:        enabled,
	}
	return u.repo.UpsertHostCallRate(ctx, rate)
}

func (u *paidInteractionUsecase) GetHostRates(ctx context.Context, hostID domain.UUID) (*domain.HostCallRate, error) {
	return u.repo.GetHostCallRate(ctx, hostID)
}

func (u *paidInteractionUsecase) RequestCall(ctx context.Context, callerID, hostID domain.UUID, callType string) (*domain.CallSession, error) {
	// Check host rate
	rate, err := u.repo.GetHostCallRate(ctx, hostID)
	if err != nil || rate == nil || !rate.IsEnabled {
		return nil, errors.New("host is not accepting calls")
	}

	currentRate := rate.VoiceCallRateIDR
	if callType == domain.CallTypeVideo {
		currentRate = rate.VideoCallRateIDR
	}

	// Check balance (min 1 minute)
	wallet, err := u.walletRepo.GetByUserID(ctx, callerID)
	if err != nil || wallet.Balance < currentRate {
		return nil, fmt.Errorf("insufficient balance. minimum required: %d IDR", currentRate)
	}

	// Create session
	session := &domain.CallSession{
		ID:       domain.NewUUIDv7(),
		HostID:   hostID,
		CallerID: callerID,
		Type:     callType,
		RateIDR:  currentRate,
		Status:   domain.CallStatusPending,
	}

	err = u.repo.CreateCallSession(ctx, session)
	if err != nil {
		return nil, err
	}

	// Set in-call status in Redis
	u.redis.Set(ctx, fmt.Sprintf("call:active:%s", hostID), session.ID.String(), 5*time.Minute)

	return session, nil
}

func (u *paidInteractionUsecase) AcceptCall(ctx context.Context, sessionID domain.UUID) error {
	session, err := u.repo.GetCallSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	now := time.Now()
	session.Status = domain.CallStatusActive
	session.StartedAt = &now
	
	return u.repo.UpdateCallSession(ctx, session)
}

func (u *paidInteractionUsecase) RejectCall(ctx context.Context, sessionID domain.UUID, reason string) error {
	session, err := u.repo.GetCallSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Status = domain.CallStatusRejected
	session.EndedReason = &reason
	
	u.redis.Del(ctx, fmt.Sprintf("call:active:%s", session.HostID))
	return u.repo.UpdateCallSession(ctx, session)
}

func (u *paidInteractionUsecase) EndCall(ctx context.Context, sessionID domain.UUID, reason string) error {
	session, err := u.repo.GetCallSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status == domain.CallStatusEnded {
		return nil
	}

	now := time.Now()
	session.Status = domain.CallStatusEnded
	session.EndedAt = &now
	session.EndedReason = &reason
	
	if session.StartedAt != nil {
		session.DurationSeconds = int(now.Sub(*session.StartedAt).Seconds())
	}

	u.redis.Del(ctx, fmt.Sprintf("call:active:%s", session.HostID))
	return u.repo.UpdateCallSession(ctx, session)
}

func (u *paidInteractionUsecase) ProcessBillingTick(ctx context.Context, sessionID domain.UUID) error {
	session, err := u.repo.GetCallSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	if session.Status != domain.CallStatusActive {
		return errors.New("session is not active")
	}

	// Calculate tick number (starting from 1)
	tickNumber := 1
	if session.StartedAt != nil {
		tickNumber = int(time.Since(*session.StartedAt).Minutes()) + 1
	}

	// Deduct balance
	err = u.walletRepo.DebitBalance(ctx, session.CallerID, session.RateIDR)
	if err != nil {
		u.EndCall(ctx, sessionID, "balance_insufficient")
		return err
	}

	// Create billing tick
	tick := &domain.CallBillingTick{
		ID:            domain.NewUUIDv7(),
		CallSessionID: sessionID,
		TickNumber:    tickNumber,
		ChargeIDR:     session.RateIDR,
	}
	u.repo.CreateBillingTick(ctx, tick)

	// Revenue Split
	// Host 60%, Platform 40% (or 40/20/40 for agency)
	hostAmount := int64(float64(session.RateIDR) * 0.6)
	platformAmount := session.RateIDR - hostAmount

	// Check agency
	rel, _ := u.agencyRepo.GetHostRelation(ctx, session.HostID)
	if rel != nil && rel.Status == domain.AgencyHostActive {
		hostAmount = int64(float64(session.RateIDR) * 0.4)
		agencyAmount := int64(float64(session.RateIDR) * 0.2)
		u.walletRepo.CreditBalance(ctx, rel.AgencyID, agencyAmount) // Simplified: agency owner wallet
	}

	u.walletRepo.CreditBalance(ctx, session.HostID, hostAmount)
	// Platform fee record could go to a special admin wallet
	
	// Update session totals
	session.TotalChargeIDR += session.RateIDR
	session.PlatformFeeIDR += platformAmount
	session.HostEarningIDR += hostAmount
	u.repo.UpdateCallSession(ctx, session)

	return nil
}

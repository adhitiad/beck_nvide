package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
	"nvide-live/pkg/redis"
)

type GiftUseCase struct {
	giftRepo    domain.GiftRepository
	giftTxRepo  domain.GiftTransactionRepository
	agencyRepo  domain.AgencyRepository
	walletUC    *WalletUseCase
	wsHub       *websocket.Hub
	redis       *redis.Client
	logger      *zap.Logger
}

func NewGiftUseCase(
	giftRepo domain.GiftRepository,
	giftTxRepo domain.GiftTransactionRepository,
	agencyRepo domain.AgencyRepository,
	walletUC *WalletUseCase,
	wsHub *websocket.Hub,
	redis *redis.Client,
	logger *zap.Logger,
) *GiftUseCase {
	return &GiftUseCase{
		giftRepo:   giftRepo,
		giftTxRepo: giftTxRepo,
		agencyRepo: agencyRepo,
		walletUC:   walletUC,
		wsHub:      wsHub,
		redis:      redis,
		logger:     logger,
	}
}

// SendGift processes a gift send with revenue split and WS broadcast
func (uc *GiftUseCase) SendGift(ctx context.Context, senderID, receiverID domain.UUID, streamID *domain.UUID, giftID domain.UUID, quantity int) (*domain.GiftTransaction, error) {
	// Rate limit: 50 gifts/min per user per stream
	if streamID != nil && uc.redis != nil {
		rlKey := fmt.Sprintf("gift:rl:%s:%s", senderID.String(), streamID.String())
		count, _ := uc.redis.GetClient().Incr(ctx, rlKey).Result()
		if count == 1 {
			uc.redis.GetClient().Expire(ctx, rlKey, 1*time.Minute)
		}
		if count > 50 {
			return nil, domain.NewDomainError(domain.ErrCodeRateLimit, "gift rate limit exceeded (50/min)", nil)
		}
	}

	// Get gift
	gift, err := uc.giftRepo.GetByID(ctx, giftID)
	if err != nil {
		return nil, err
	}
	if !gift.IsActive {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "gift is not active", nil)
	}

	totalPrice := gift.Price * int64(quantity)
	idempotencyKey := fmt.Sprintf("gift:%s:%s:%s:%d", senderID.String(), giftID.String(), time.Now().Format("20060102150405"), quantity)

	// Debit sender wallet
	if err := uc.walletUC.DebitWallet(ctx, senderID, totalPrice, domain.TxTypeGiftSent, idempotencyKey); err != nil {
		return nil, err
	}

	// Determine revenue split
	hasAgency := false
	var agencyID *domain.UUID
	agencyCommissionRate := 25 // default 25% of 80% pool
	agencyRelation, err := uc.agencyRepo.GetHostRelation(ctx, receiverID)
	if err == nil && agencyRelation != nil {
		hasAgency = true
		agencyID = &agencyRelation.AgencyID
		agencyCommissionRate = 100 - agencyRelation.RevenueShare // e.g. if host share is 60, agency gets 40 of remaining
		// But let's use the fixed rule: Platform 20%, then split remaining
		// Actually let's use the agency's commission_rate
		agency, _ := uc.agencyRepo.GetByID(ctx, agencyRelation.AgencyID)
		if agency != nil {
			agencyCommissionRate = agency.CommissionRate
		}
	}

	split := CalculateRevenueSplit(totalPrice, hasAgency, agencyCommissionRate)

	// Credit receiver (host)
	uc.walletUC.CreditWallet(ctx, receiverID, split.HostEarning, domain.TxTypeHostEarning, idempotencyKey+":host")

	// Credit agency if applicable
	if hasAgency && agencyID != nil {
		agency, _ := uc.agencyRepo.GetByID(ctx, *agencyID)
		if agency != nil {
			uc.walletUC.CreditWallet(ctx, agency.OwnerID, split.AgencyCommission, domain.TxTypeAgencyCommission, idempotencyKey+":agency")
			uc.agencyRepo.UpdateHostEarnings(ctx, *agencyID, receiverID, split.HostEarning)
		}
	}

	// Record gift transaction
	gtx := &domain.GiftTransaction{
		ID:               domain.NewUUID(),
		StreamID:         streamID,
		SenderID:         senderID,
		ReceiverID:       receiverID,
		GiftID:           giftID,
		Quantity:         quantity,
		TotalPrice:       totalPrice,
		AgencyID:         agencyID,
		AgencyCommission: split.AgencyCommission,
		HostEarning:      split.HostEarning,
		PlatformFee:      split.PlatformFee,
	}
	if err := uc.giftTxRepo.Create(ctx, gtx); err != nil {
		uc.logger.Error("Failed to record gift transaction", zap.Error(err))
	}

	// Update leaderboard in Redis
	if streamID != nil && uc.redis != nil {
		lbKey := fmt.Sprintf("gift:leaderboard:%s", streamID.String())
		uc.redis.GetClient().ZIncrBy(ctx, lbKey, float64(totalPrice), senderID.String())
	}

	// Broadcast to stream room
	if streamID != nil && uc.wsHub != nil {
		payload := map[string]interface{}{
			"type": "gift",
			"payload": map[string]interface{}{
				"gift_name":     gift.Name,
				"gift_icon":     gift.IconURL,
				"animation_url": gift.AnimationURL,
				"quantity":      quantity,
				"total_price":   totalPrice,
				"sender_id":     senderID.String(),
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}
		data, _ := json.Marshal(payload)
		uc.wsHub.BroadcastToRoom(streamID.String(), data)
	}

	return gtx, nil
}

// GetGiftCatalog returns active gifts
func (uc *GiftUseCase) GetGiftCatalog(ctx context.Context) ([]*domain.Gift, error) {
	return uc.giftRepo.ListActive(ctx)
}

// GetLeaderboard returns top 10 gifters for a stream
func (uc *GiftUseCase) GetLeaderboard(ctx context.Context, streamID domain.UUID) ([]map[string]interface{}, error) {
	if uc.redis == nil {
		return nil, nil
	}
	lbKey := fmt.Sprintf("gift:leaderboard:%s", streamID.String())
	results, err := uc.redis.GetClient().ZRevRangeWithScores(ctx, lbKey, 0, 9).Result()
	if err != nil {
		return nil, err
	}

	var leaderboard []map[string]interface{}
	for rank, z := range results {
		leaderboard = append(leaderboard, map[string]interface{}{
			"rank":    rank + 1,
			"user_id": z.Member,
			"total":   int64(z.Score),
		})
	}
	return leaderboard, nil
}

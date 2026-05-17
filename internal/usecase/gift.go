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
	privateUC   domain.PrivateChatUsecase
	messageRepo domain.MessageRepository
	chatRepo    domain.ChatRoomRepository
	wsHub       *websocket.Hub
	redis       *redis.Client
	logger      *zap.Logger
}

func NewGiftUseCase(
	giftRepo domain.GiftRepository,
	giftTxRepo domain.GiftTransactionRepository,
	agencyRepo domain.AgencyRepository,
	walletUC *WalletUseCase,
	privateUC domain.PrivateChatUsecase,
	messageRepo domain.MessageRepository,
	chatRepo domain.ChatRoomRepository,
	wsHub *websocket.Hub,
	redis *redis.Client,
	logger *zap.Logger,
) *GiftUseCase {
	return &GiftUseCase{
		giftRepo:    giftRepo,
		giftTxRepo:  giftTxRepo,
		agencyRepo:  agencyRepo,
		walletUC:    walletUC,
		privateUC:   privateUC,
		messageRepo: messageRepo,
		chatRepo:    chatRepo,
		wsHub:       wsHub,
		redis:       redis,
		logger:      logger,
	}
}

// SendGift processes a gift send with revenue split and WS broadcast
func (uc *GiftUseCase) SendGift(ctx context.Context, senderID, receiverID domain.UUID, roomID *domain.UUID, giftID domain.UUID, quantity int) (*domain.GiftTransaction, error) {
	// 1. Get gift details
	gift, err := uc.giftRepo.GetByID(ctx, giftID)
	if err != nil {
		return nil, err
	}
	if !gift.IsActive {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "gift is not active", nil)
	}

	totalPrice := gift.Price * int64(quantity)
	idempotencyKey := fmt.Sprintf("gift:%s:%s:%s:%d", senderID.String(), giftID.String(), time.Now().Format("20060102150405"), quantity)

	// 2. Debit sender wallet
	if err := uc.walletUC.DebitWallet(ctx, senderID, totalPrice, domain.TxTypeGiftSent, idempotencyKey); err != nil {
		return nil, err
	}

	// 3. Determine revenue split
	hasAgency := false
	var agencyID *domain.UUID
	agencyCommissionRate := 25 
	agencyRelation, err := uc.agencyRepo.GetHostRelation(ctx, receiverID)
	if err == nil && agencyRelation != nil {
		hasAgency = true
		agencyID = &agencyRelation.AgencyID
		agency, _ := uc.agencyRepo.GetByID(ctx, agencyRelation.AgencyID)
		if agency != nil {
			agencyCommissionRate = agency.CommissionRate
		}
	}

	split := CalculateRevenueSplit(totalPrice, hasAgency, agencyCommissionRate)

	// 4. Credit receiver (host) & agency
	uc.walletUC.CreditWallet(ctx, receiverID, split.HostEarning, domain.TxTypeHostEarning, idempotencyKey+":host")
	if hasAgency && agencyID != nil {
		agency, _ := uc.agencyRepo.GetByID(ctx, *agencyID)
		if agency != nil {
			uc.walletUC.CreditWallet(ctx, agency.OwnerID, split.AgencyCommission, domain.TxTypeAgencyCommission, idempotencyKey+":agency")
		}
	}

	// 5. Record gift transaction
	var streamID *domain.UUID
	if roomID != nil {
		room, err := uc.chatRepo.GetByID(ctx, *roomID)
		if err == nil && room.Type == "stream" {
			streamID = room.TargetID
		}
	}

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
	_ = uc.giftTxRepo.Create(ctx, gtx)

	// 6. Combo & PK Battle Calculations
	comboCount := int64(1)
	comboTier := 1
	isPKVote := false
	isCritical := false
	pkPoints := totalPrice

	if roomID != nil && uc.redis != nil {
		// Combo tracking via Redis
		comboKey := fmt.Sprintf("gift:combo:%s:%s:%s", roomID.String(), senderID.String(), giftID.String())
		var incrErr error
		comboCount, incrErr = uc.redis.GetClient().Incr(ctx, comboKey).Result()
		if incrErr == nil {
			uc.redis.GetClient().Expire(ctx, comboKey, 5*time.Second)
		} else {
			comboCount = 1
		}

		if comboCount >= 9999 {
			comboTier = 7
		} else if comboCount >= 3344 {
			comboTier = 6
		} else if comboCount >= 1314 {
			comboTier = 5
		} else if comboCount >= 520 {
			comboTier = 4
		} else if comboCount >= 188 {
			comboTier = 3
		} else if comboCount >= 66 {
			comboTier = 2
		}

		// PK Battle Scoring, Gift Velocity, and Leveling XP
		if streamID != nil {
			// 1. Gift Velocity tracking (5 min TTL)
			giftVelocityKey := fmt.Sprintf("stream:gift_velocity:%s", streamID.String())
			uc.redis.GetClient().IncrBy(ctx, giftVelocityKey, totalPrice)
			uc.redis.GetClient().Expire(ctx, giftVelocityKey, 5*time.Minute)

			// 2. Batch XP queuing (1 IDR = 1 XP)
			uc.redis.GetClient().HIncrBy(ctx, "xp:batch:user", senderID.String(), totalPrice)
			uc.redis.GetClient().HIncrBy(ctx, "xp:batch:host", receiverID.String(), totalPrice)

			// 3. PK Battle calculation
			activePKKey := fmt.Sprintf("stream:active_pk:%s", streamID.String())
			pkIDStr, err := uc.redis.GetClient().Get(ctx, activePKKey).Result()
			if err == nil && pkIDStr != "" {
				isPKVote = true
				if time.Now().UnixNano()%100 < 3 {
					pkPoints = pkPoints * 2
					isCritical = true
				}
				scoresKey := fmt.Sprintf("pk:battle:scores:%s", pkIDStr)
				uc.redis.GetClient().ZIncrBy(ctx, scoresKey, float64(pkPoints), receiverID.String())
			}
		}
	}

	// 7. Broadcast via WebSocket and Save as Message
	if roomID != nil {
		content := fmt.Sprintf("🎁 Mengirim %d %s", quantity, gift.Name)
		msg := &domain.Message{
			ID:      domain.NewUUID(),
			RoomID:  *roomID,
			UserID:  senderID,
			Content: content,
			Type:    "gift",
		}
		_ = uc.messageRepo.Create(ctx, msg)

		if uc.wsHub != nil {
			payload, _ := json.Marshal(map[string]interface{}{
				"type": "gift",
				"payload": map[string]interface{}{
					"gift_id":       giftID,
					"gift_name":     gift.Name,
					"gift_icon":     gift.IconURL,
					"animation_url": gift.AnimationURL,
					"quantity":      quantity,
					"sender_id":     senderID.String(),
					"room_id":       roomID.String(),
					"combo_count":   comboCount,
					"combo_tier":    comboTier,
					"is_pk_vote":    isPKVote,
					"pk_points":     pkPoints,
					"is_critical":   isCritical,
				},
				"timestamp": time.Now().Format(time.RFC3339),
			})
			uc.wsHub.BroadcastToRoom(roomID.String(), payload)
		}
	}

	return gtx, nil
}

// SendPrivateGift processes a gift in private chat
func (uc *GiftUseCase) SendPrivateGift(ctx context.Context, senderID, convID domain.UUID, giftID domain.UUID, quantity int) (*domain.GiftTransaction, error) {
	// 1. Get Conversation to find the receiver
	conv, err := uc.privateUC.GetConversationByID(ctx, convID)
	if err != nil {
		return nil, err
	}

	// 2. Determine ReceiverID (the other person in the conversation)
	receiverID := conv.RecipientID
	if receiverID == senderID {
		receiverID = conv.InitiatorID
	}

	// 3. Call SendGift with roomID (which is the convID)
	return uc.SendGift(ctx, senderID, receiverID, &convID, giftID, quantity)
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

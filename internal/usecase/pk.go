package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
	"nvide-live/pkg/redis"
)

type PKBattleUseCase struct {
	pkRepo     domain.PKBattleRepository
	streamRepo domain.StreamRepository
	wsHub      *websocket.Hub
	redis      *redis.Client
	logger     *zap.Logger
}

func NewPKBattleUseCase(
	pkRepo domain.PKBattleRepository,
	streamRepo domain.StreamRepository,
	wsHub *websocket.Hub,
	redis *redis.Client,
	logger *zap.Logger,
) *PKBattleUseCase {
	return &PKBattleUseCase{
		pkRepo:     pkRepo,
		streamRepo: streamRepo,
		wsHub:      wsHub,
		redis:      redis,
		logger:     logger,
	}
}

// InvitePKBattle invites another host to a PK Battle
func (uc *PKBattleUseCase) InvitePKBattle(ctx context.Context, hostAID, hostBID domain.UUID) (*domain.PKBattle, error) {
	// Verify host A is live
	streamA, err := uc.streamRepo.GetLiveByHost(ctx, hostAID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "host A is not streaming live", nil)
	}

	// Verify host B is live
	streamB, err := uc.streamRepo.GetLiveByHost(ctx, hostBID)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "target host is not streaming live", nil)
	}

	// Check if already in active/invite PK
	existing, err := uc.pkRepo.GetActiveByHost(ctx, hostAID)
	if err == nil && existing != nil {
		return nil, domain.NewDomainError(domain.ErrCodeConflict, "host is already in a PK session", nil)
	}

	pk := &domain.PKBattle{
		ID:        domain.NewUUID(),
		StreamAID: streamA.ID,
		StreamBID: streamB.ID,
		HostAID:   hostAID,
		HostBID:   hostBID,
		Status:    "invite",
	}

	if err := uc.pkRepo.Create(ctx, pk); err != nil {
		return nil, err
	}

	// Notify Host B via websocket
	if uc.wsHub != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"type": "pk_invite",
			"payload": map[string]interface{}{
				"pk_id":         pk.ID.String(),
				"host_a_id":     hostAID.String(),
				"stream_a_id":   streamA.ID.String(),
				"host_a_name":   streamA.Title, // using stream title as proxy name
				"thumbnail_url": streamA.ThumbnailURL,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
		uc.wsHub.BroadcastToRoom(streamB.RoomID.String(), payload)
	}

	return pk, nil
}

// AcceptPKBattle accepts the PK Battle invitation
func (uc *PKBattleUseCase) AcceptPKBattle(ctx context.Context, hostBID, pkID domain.UUID) (*domain.PKBattle, error) {
	pk, err := uc.pkRepo.GetByID(ctx, pkID)
	if err != nil {
		return nil, err
	}

	if pk.HostBID != hostBID {
		return nil, domain.NewDomainError(domain.ErrCodeForbidden, "only the invited host can accept this PK battle", nil)
	}

	if pk.Status != "invite" {
		return nil, domain.NewDomainError(domain.ErrCodeValidation, "PK battle invitation has already expired or been accepted", nil)
	}

	pk.Status = "active"
	now := time.Now()
	pk.StartedAt = &now

	if err := uc.pkRepo.Update(ctx, pk); err != nil {
		return nil, err
	}

	if uc.redis != nil {
		// Store active PK keys in Redis for fast access in gift flow
		uc.redis.GetClient().Set(ctx, fmt.Sprintf("stream:active_pk:%s", pk.StreamAID.String()), pk.ID.String(), 10*time.Minute)
		uc.redis.GetClient().Set(ctx, fmt.Sprintf("stream:active_pk:%s", pk.StreamBID.String()), pk.ID.String(), 10*time.Minute)

		// Set initial scores
		scoresKey := fmt.Sprintf("pk:battle:scores:%s", pk.ID.String())
		uc.redis.GetClient().ZAdd(ctx, scoresKey,
			goredis.Z{Score: 0, Member: pk.HostAID.String()},
			goredis.Z{Score: 0, Member: pk.HostBID.String()},
		)
	}

	// Fetch streams to broadcast to their WebRTC Rooms
	streamA, _ := uc.streamRepo.GetByID(ctx, pk.StreamAID)
	streamB, _ := uc.streamRepo.GetByID(ctx, pk.StreamBID)

	if uc.wsHub != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"type": "pk_start",
			"payload": map[string]interface{}{
				"pk_id":       pk.ID.String(),
				"host_a_id":   pk.HostAID.String(),
				"host_b_id":   pk.HostBID.String(),
				"stream_a_id": pk.StreamAID.String(),
				"stream_b_id": pk.StreamBID.String(),
				"duration":    300,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
		if streamA != nil {
			uc.wsHub.BroadcastToRoom(streamA.RoomID.String(), payload)
		}
		if streamB != nil {
			uc.wsHub.BroadcastToRoom(streamB.RoomID.String(), payload)
		}
	}

	return pk, nil
}

// RejectPKBattle rejects the PK Battle invitation
func (uc *PKBattleUseCase) RejectPKBattle(ctx context.Context, hostBID, pkID domain.UUID) error {
	pk, err := uc.pkRepo.GetByID(ctx, pkID)
	if err != nil {
		return err
	}

	if pk.HostBID != hostBID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "only the invited host can reject this PK battle", nil)
	}

	pk.Status = "rejected"
	if err := uc.pkRepo.Update(ctx, pk); err != nil {
		return err
	}

	streamA, _ := uc.streamRepo.GetByID(ctx, pk.StreamAID)
	if streamA != nil && uc.wsHub != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"type": "pk_rejected",
			"payload": map[string]interface{}{
				"pk_id":     pk.ID.String(),
				"host_b_id": hostBID.String(),
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
		uc.wsHub.BroadcastToRoom(streamA.RoomID.String(), payload)
	}

	return nil
}

// GetPKStatus gets current scores and states with inline self-healing state transitions
func (uc *PKBattleUseCase) GetPKStatus(ctx context.Context, pkID domain.UUID) (map[string]interface{}, error) {
	pk, err := uc.pkRepo.GetByID(ctx, pkID)
	if err != nil {
		return nil, err
	}

	// Trigger self-healing state updates based on time elapsed
	uc.checkAndUpdateStatus(ctx, pk)

	// Fetch current scores from Redis if active
	scoreA := pk.ScoreA
	scoreB := pk.ScoreB
	if uc.redis != nil && (pk.Status == "active" || pk.Status == "punishment") {
		scoresKey := fmt.Sprintf("pk:battle:scores:%s", pk.ID.String())
		valA, _ := uc.redis.GetClient().ZScore(ctx, scoresKey, pk.HostAID.String()).Result()
		valB, _ := uc.redis.GetClient().ZScore(ctx, scoresKey, pk.HostBID.String()).Result()
		scoreA = int64(valA)
		scoreB = int64(valB)
	}

	timeLeft := int64(0)
	if pk.Status == "active" && pk.StartedAt != nil {
		elapsed := time.Since(*pk.StartedAt)
		timeLeft = int64((5 * time.Minute) / time.Second) - int64(elapsed/time.Second)
		if timeLeft < 0 {
			timeLeft = 0
		}
	} else if pk.Status == "punishment" && pk.PunishmentStart != nil {
		elapsed := time.Since(*pk.PunishmentStart)
		timeLeft = int64((2 * time.Minute) / time.Second) - int64(elapsed/time.Second)
		if timeLeft < 0 {
			timeLeft = 0
		}
	}

	var winnerIDStr string
	if pk.WinnerID != nil {
		winnerIDStr = pk.WinnerID.String()
	}

	return map[string]interface{}{
		"pk_id":             pk.ID.String(),
		"status":            pk.Status,
		"host_a_id":         pk.HostAID.String(),
		"host_b_id":         pk.HostBID.String(),
		"score_a":           scoreA,
		"score_b":           scoreB,
		"winner_id":         winnerIDStr,
		"time_left_seconds": timeLeft,
	}, nil
}

func (uc *PKBattleUseCase) checkAndUpdateStatus(ctx context.Context, pk *domain.PKBattle) {
	if pk.Status == "active" && pk.StartedAt != nil {
		elapsed := time.Since(*pk.StartedAt)
		if elapsed >= 5*time.Minute {
			pk.Status = "punishment"
			now := time.Now()
			pk.PunishmentStart = &now

			// Determine winner
			scoreA := int64(0)
			scoreB := int64(0)
			if uc.redis != nil {
				scoresKey := fmt.Sprintf("pk:battle:scores:%s", pk.ID.String())
				valA, _ := uc.redis.GetClient().ZScore(ctx, scoresKey, pk.HostAID.String()).Result()
				valB, _ := uc.redis.GetClient().ZScore(ctx, scoresKey, pk.HostBID.String()).Result()
				scoreA = int64(valA)
				scoreB = int64(valB)
			}
			pk.ScoreA = scoreA
			pk.ScoreB = scoreB

			if scoreA > scoreB {
				pk.WinnerID = &pk.HostAID
			} else if scoreB > scoreA {
				pk.WinnerID = &pk.HostBID
			}

			_ = uc.pkRepo.Update(ctx, pk)
			uc.broadcastPKStatus(pk)
		}
	} else if pk.Status == "punishment" && pk.PunishmentStart != nil {
		elapsed := time.Since(*pk.PunishmentStart)
		if elapsed >= 2*time.Minute {
			pk.Status = "ended"
			now := time.Now()
			pk.EndedAt = &now
			_ = uc.pkRepo.Update(ctx, pk)

			// Clean Redis keys
			if uc.redis != nil {
				uc.redis.GetClient().Del(ctx, fmt.Sprintf("stream:active_pk:%s", pk.StreamAID.String()))
				uc.redis.GetClient().Del(ctx, fmt.Sprintf("stream:active_pk:%s", pk.StreamBID.String()))
			}

			uc.broadcastPKStatus(pk)
		}
	}
}

func (uc *PKBattleUseCase) broadcastPKStatus(pk *domain.PKBattle) {
	if uc.wsHub == nil {
		return
	}

	var winnerIDStr string
	if pk.WinnerID != nil {
		winnerIDStr = pk.WinnerID.String()
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"type": "pk_status_change",
		"payload": map[string]interface{}{
			"pk_id":     pk.ID.String(),
			"status":    pk.Status,
			"score_a":   pk.ScoreA,
			"score_b":   pk.ScoreB,
			"winner_id": winnerIDStr,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})

	streamA, _ := uc.streamRepo.GetByID(context.Background(), pk.StreamAID)
	streamB, _ := uc.streamRepo.GetByID(context.Background(), pk.StreamBID)

	if streamA != nil {
		uc.wsHub.BroadcastToRoom(streamA.RoomID.String(), payload)
	}
	if streamB != nil {
		uc.wsHub.BroadcastToRoom(streamB.RoomID.String(), payload)
	}
}

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/internal/websocket"
	"nvide-live/pkg/redis"
)

type LevelingUseCase struct {
	db     *pgxpool.Pool
	redis  *redis.Client
	wsHub  *websocket.Hub
	logger *zap.Logger
}

func NewLevelingUseCase(
	db *pgxpool.Pool,
	redis *redis.Client,
	wsHub *websocket.Hub,
	logger *zap.Logger,
) *LevelingUseCase {
	return &LevelingUseCase{
		db:     db,
		redis:  redis,
		wsHub:  wsHub,
		logger: logger,
	}
}

// TrackWatchTimeXP queues XP for user and host for watch time
// User receives +5 XP per 1 minute, Host receives +10 XP per 1 minute
func (uc *LevelingUseCase) TrackWatchTimeXP(ctx context.Context, userID, hostID domain.UUID, minutes int) error {
	if uc.redis == nil {
		return nil
	}
	userXP := int64(minutes * 5)
	hostXP := int64(minutes * 10)

	uc.redis.GetClient().HIncrBy(ctx, "xp:batch:user", userID.String(), userXP)
	uc.redis.GetClient().HIncrBy(ctx, "xp:batch:host", hostID.String(), hostXP)
	return nil
}

// FlushXPUpdates flushes queued XP updates in Redis Hash to PostgreSQL in an atomic manner
func (uc *LevelingUseCase) FlushXPUpdates(ctx context.Context) error {
	if uc.redis == nil {
		return nil
	}

	// 1. Fetch and clear user XP batch
	userBatch, err := uc.redis.GetClient().HGetAll(ctx, "xp:batch:user").Result()
	if err == nil && len(userBatch) > 0 {
		uc.redis.GetClient().Del(ctx, "xp:batch:user")
		for userIDStr, xpStr := range userBatch {
			userID, err := domain.FromString(userIDStr)
			if err != nil {
				continue
			}
			xpAmount, _ := strconv.ParseInt(xpStr, 10, 64)
			if xpAmount > 0 {
				go uc.updateUserXP(ctx, userID, xpAmount)
			}
		}
	}

	// 2. Fetch and clear host XP batch
	hostBatch, err := uc.redis.GetClient().HGetAll(ctx, "xp:batch:host").Result()
	if err == nil && len(hostBatch) > 0 {
		uc.redis.GetClient().Del(ctx, "xp:batch:host")
		for hostIDStr, xpStr := range hostBatch {
			hostID, err := domain.FromString(hostIDStr)
			if err != nil {
				continue
			}
			xpAmount, _ := strconv.ParseInt(xpStr, 10, 64)
			if xpAmount > 0 {
				go uc.updateHostXP(ctx, hostID, xpAmount)
			}
		}
	}

	return nil
}

func (uc *LevelingUseCase) updateUserXP(ctx context.Context, userID domain.UUID, xp int64) {
	// Query current level to detect level ups
	var oldLevel int
	err := uc.db.QueryRow(ctx, "SELECT user_level FROM users WHERE id = $1", userID).Scan(&oldLevel)
	if err != nil {
		return
	}

	// Update user XP & Level atomically
	updateSQL := `
		UPDATE users
		SET user_xp = user_xp + $1,
			user_level = COALESCE((SELECT level FROM user_levels WHERE min_xp <= users.user_xp + $1 ORDER BY level DESC LIMIT 1), 1),
			updated_at = NOW()
		WHERE id = $2
		RETURNING user_xp, user_level
	`
	var newXP int64
	var newLevel int
	err = uc.db.QueryRow(ctx, updateSQL, xp, userID).Scan(&newXP, &newLevel)
	if err != nil {
		uc.logger.Error("Failed to update user XP in Postgres", zap.Error(err), zap.String("user_id", userID.String()))
		return
	}

	// Insert UserXPLog record
	logSQL := `
		INSERT INTO user_xp_logs (id, user_id, action, xp_amount, created_at)
		VALUES (gen_random_uuid(), $1, 'gift_or_watch', $2, NOW())
	`
	_, _ = uc.db.Exec(ctx, logSQL, userID, xp)

	// Trigger Level Up effect if increased
	if newLevel > oldLevel {
		uc.logger.Info("User Leveled Up!", zap.String("user_id", userID.String()), zap.Int("old", oldLevel), zap.Int("new", newLevel))
		uc.broadcastLevelUp(userID, newLevel, false)
	}
}

func (uc *LevelingUseCase) updateHostXP(ctx context.Context, hostID domain.UUID, xp int64) {
	// Query current level to detect level ups
	var oldLevel int
	err := uc.db.QueryRow(ctx, "SELECT host_level FROM users WHERE id = $1", hostID).Scan(&oldLevel)
	if err != nil {
		return
	}

	// Update host XP & Level atomically
	updateSQL := `
		UPDATE users
		SET host_xp = host_xp + $1,
			host_level = COALESCE((SELECT level FROM host_levels WHERE min_xp <= users.host_xp + $1 ORDER BY level DESC LIMIT 1), 1),
			updated_at = NOW()
		WHERE id = $2
		RETURNING host_xp, host_level
	`
	var newXP int64
	var newLevel int
	err = uc.db.QueryRow(ctx, updateSQL, xp, hostID).Scan(&newXP, &newLevel)
	if err != nil {
		uc.logger.Error("Failed to update host XP in Postgres", zap.Error(err), zap.String("host_id", hostID.String()))
		return
	}

	// Insert UserXPLog record for host
	logSQL := `
		INSERT INTO user_xp_logs (id, user_id, action, xp_amount, created_at)
		VALUES (gen_random_uuid(), $1, 'host_gift_received', $2, NOW())
	`
	_, _ = uc.db.Exec(ctx, logSQL, hostID, xp)

	// Trigger Level Up effect if increased
	if newLevel > oldLevel {
		uc.logger.Info("Host Leveled Up!", zap.String("host_id", hostID.String()), zap.Int("old", oldLevel), zap.Int("new", newLevel))
		uc.broadcastLevelUp(hostID, newLevel, true)
	}
}

func (uc *LevelingUseCase) broadcastLevelUp(userID domain.UUID, newLevel int, isHost bool) {
	if uc.wsHub == nil {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"type": "level_up",
		"payload": map[string]interface{}{
			"user_id":   userID.String(),
			"new_level": newLevel,
			"is_host":   isHost,
			"message":   fmt.Sprintf("🎉 Selamat! Kamu naik ke level %d!", newLevel),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})

	uc.wsHub.BroadcastToRoom("system", payload)
}

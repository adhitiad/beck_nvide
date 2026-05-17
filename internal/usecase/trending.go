package usecase

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

type TrendingUseCase struct {
	db         *pgxpool.Pool
	redis      *redis.Client
	streamRepo domain.StreamRepository
	logger     *zap.Logger
}

func NewTrendingUseCase(
	db *pgxpool.Pool,
	redis *redis.Client,
	streamRepo domain.StreamRepository,
	logger *zap.Logger,
) *TrendingUseCase {
	return &TrendingUseCase{
		db:         db,
		redis:      redis,
		streamRepo: streamRepo,
		logger:     logger,
	}
}

// RecalculateTrendingScores computes active streams' trending scores and caches them in Redis Sorted Set
// Score = ViewerCount * 10 + GiftVelocity * 5 + PKActiveBoost (100)
func (uc *TrendingUseCase) RecalculateTrendingScores(ctx context.Context) error {
	if uc.redis == nil {
		return nil
	}

	query := `
		SELECT id, viewer_count, category
		FROM streams
		WHERE status = 'live'
	`
	rows, err := uc.db.Query(ctx, query)
	if err != nil {
		uc.logger.Error("Failed to query live streams for trending recalculation", zap.Error(err))
		return err
	}
	defer rows.Close()

	var zMembers []goredis.Z
	for rows.Next() {
		var streamIDStr string
		var viewerCount int
		var category string
		err := rows.Scan(&streamIDStr, &viewerCount, &category)
		if err != nil {
			continue
		}

		// Calculate score
		score := float64(viewerCount * 10)

		// 1. Gift velocity check (value sent in last 5 minutes)
		giftVelocityKey := fmt.Sprintf("stream:gift_velocity:%s", streamIDStr)
		velocityStr, err := uc.redis.GetClient().Get(ctx, giftVelocityKey).Result()
		if err == nil && velocityStr != "" {
			velocity, _ := strconv.ParseInt(velocityStr, 10, 64)
			score += float64(velocity) * 5
		}

		// 2. PK battle active boost (+100 points)
		activePKKey := fmt.Sprintf("stream:active_pk:%s", streamIDStr)
		pkExists, _ := uc.redis.GetClient().Exists(ctx, activePKKey).Result()
		if pkExists > 0 {
			score += 100
		}

		zMembers = append(zMembers, goredis.Z{
			Score:  score,
			Member: streamIDStr,
		})
	}

	trendingKey := "trending:global"
	if len(zMembers) > 0 {
		uc.redis.GetClient().Del(ctx, trendingKey)
		
		err = uc.redis.GetClient().ZAdd(ctx, trendingKey, zMembers...).Err()
		if err != nil {
			uc.logger.Error("Failed to ZAdd trending scores to Redis", zap.Error(err))
			return err
		}
	} else {
		uc.redis.GetClient().Del(ctx, trendingKey)
	}

	return nil
}

// GetTrendingStreams fetches streams ordered by their trending score in Redis
func (uc *TrendingUseCase) GetTrendingStreams(ctx context.Context, limit int) ([]*domain.Stream, error) {
	if uc.redis == nil {
		return uc.streamRepo.ListLive(ctx, limit, 0)
	}

	trendingKey := "trending:global"
	streamIDs, err := uc.redis.GetClient().ZRevRange(ctx, trendingKey, 0, int64(limit-1)).Result()
	if err != nil || len(streamIDs) == 0 {
		return uc.streamRepo.ListLive(ctx, limit, 0)
	}

	streams := make([]*domain.Stream, 0)
	for _, idStr := range streamIDs {
		id, err := domain.FromString(idStr)
		if err != nil {
			continue
		}
		stream, err := uc.streamRepo.GetByID(ctx, id)
		if err == nil && stream != nil && stream.Status == "live" {
			streams = append(streams, stream)
		}
	}

	return streams, nil
}

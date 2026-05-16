package usecase

import (
	"context"

	"encoding/json"
	"fmt"
	"strconv"

	"go.uber.org/zap"
	"nvide-live/internal/domain"
	"nvide-live/pkg/broker"
	"nvide-live/pkg/redis"
)

// LikeUseCase handles like business logic
type LikeUseCase struct {
	likeRepo    domain.LikeRepository
	redisClient *redis.Client
	broker      broker.Broker
	logger      *zap.Logger
}

// NewLikeUseCase creates new like usecase
func NewLikeUseCase(
	likeRepo domain.LikeRepository,
	redisClient *redis.Client,
	b broker.Broker,
	logger *zap.Logger,
) *LikeUseCase {
	return &LikeUseCase{
		likeRepo:    likeRepo,
		redisClient: redisClient,
		broker:      b,
		logger:      logger,
	}
}

// LikeContent likes content
func (uc *LikeUseCase) LikeContent(ctx context.Context, userID, contentID domain.UUID, contentType string) error {
	hasLiked, err := uc.likeRepo.HasLiked(ctx, userID, contentID, contentType)
	if err != nil {
		return err
	}

	if hasLiked {
		return domain.NewDomainError(domain.ErrCodeConflict, "already liked", nil)
	}

	like := &domain.Like{
		ID:          domain.NewUUID(),
		UserID:      userID,
		ContentID:   contentID,
		ContentType: contentType,
	}

	if err := uc.likeRepo.Create(ctx, like); err != nil {
		return err
	}

	// Increment Redis counter
	key := fmt.Sprintf("likes:count:%s:%s", contentType, contentID.String())
	newCount, err := uc.redisClient.GetClient().Incr(ctx, key).Result()
	if err != nil {
		uc.logger.Error("Failed to increment redis like count", zap.Error(err))
	} else {
		// Broadcast if it's a stream
		if contentType == "stream" && uc.broker != nil {
			msg := map[string]interface{}{
				"type": "like_update",
				"payload": map[string]interface{}{
					"content_id": contentID.String(),
					"like_count": newCount,
				},
				"timestamp": "", // let client/hub handle it
			}
			msgBytes, _ := json.Marshal(msg)
			uc.broker.Publish(ctx, "room:"+contentID.String(), string(msgBytes))
		}
	}

	uc.logger.Info("Content liked",
		zap.String("user_id", userID.String()),
		zap.String("content_id", contentID.String()),
		zap.String("content_type", contentType),
	)
	return nil
}

// UnlikeContent unlikes content
func (uc *LikeUseCase) UnlikeContent(ctx context.Context, userID, contentID domain.UUID, contentType string) error {
	return uc.likeRepo.Delete(ctx, userID, contentID, contentType)
}

// HasLiked checks if user has liked content
func (uc *LikeUseCase) HasLiked(ctx context.Context, userID, contentID domain.UUID, contentType string) (bool, error) {
	return uc.likeRepo.HasLiked(ctx, userID, contentID, contentType)
}

// GetLikeCount gets like count for content
func (uc *LikeUseCase) GetLikeCount(ctx context.Context, contentID domain.UUID, contentType string) (int, error) {
	key := fmt.Sprintf("likes:count:%s:%s", contentType, contentID.String())
	val, err := uc.redisClient.GetClient().Get(ctx, key).Result()
	if err == nil {
		count, err := strconv.Atoi(val)
		if err == nil {
			return count, nil
		}
	}

	// Fallback to DB
	count, err := uc.likeRepo.CountByContent(ctx, contentID, contentType)
	if err == nil {
		// Update cache
		uc.redisClient.GetClient().Set(ctx, key, count, 0)
	}
	return count, err
}

// GetUserLikes gets likes by user
func (uc *LikeUseCase) GetUserLikes(ctx context.Context, userID domain.UUID, limit, offset int) ([]*domain.Like, error) {
	return uc.likeRepo.GetByUser(ctx, userID, limit, offset)
}

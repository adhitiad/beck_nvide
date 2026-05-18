package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"nvide-live/internal/domain"
	"nvide-live/pkg/redis"
)

// RedisCache wraps Redis client for social features optimization
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisSocialCache creates a new RedisCache instance
func NewRedisSocialCache(client *redis.Client, logger *zap.Logger) *RedisCache {
	return &RedisCache{
		client: client,
		logger: logger,
	}
}

// -------------------------------------------------------------
// STORY VIEW COUNT CACHING (INCR & SYNC QUEUE)
// -------------------------------------------------------------

// IncrementStoryView increments the view count of a story in Redis and queues it for DB syncing
func (c *RedisCache) IncrementStoryView(ctx context.Context, storyID string) (int64, error) {
	rdb := c.client.GetClient()
	
	// Increment view count in Redis
	viewKey := fmt.Sprintf("story:views:%s", storyID)
	count, err := rdb.Incr(ctx, viewKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment story view in redis: %w", err)
	}

	// Add to set of stories that need to be synced to DB
	syncQueueKey := "stories:sync_views"
	if err := rdb.SAdd(ctx, syncQueueKey, storyID).Err(); err != nil {
		c.logger.Warn("Failed to add story ID to view sync queue", zap.Error(err), zap.String("story_id", storyID))
	}

	return count, nil
}

// GetStoryViewCount gets the cached view count from Redis
func (c *RedisCache) GetStoryViewCount(ctx context.Context, storyID string) (int64, error) {
	rdb := c.client.GetClient()
	viewKey := fmt.Sprintf("story:views:%s", storyID)
	
	val, err := rdb.Get(ctx, viewKey).Result()
	if err == goredis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

// GetAndResetStoryViews atomically retrieves the view count and resets it to "0" to prevent loss
func (c *RedisCache) GetAndResetStoryViews(ctx context.Context, storyID string) (int64, error) {
	rdb := c.client.GetClient()
	viewKey := fmt.Sprintf("story:views:%s", storyID)

	val, err := rdb.GetSet(ctx, viewKey, "0").Result()
	if err == goredis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

// GetStoriesToSyncViews retrieves all story IDs that need view count sync
func (c *RedisCache) GetStoriesToSyncViews(ctx context.Context) ([]string, error) {
	rdb := c.client.GetClient()
	syncQueueKey := "stories:sync_views"
	return rdb.SMembers(ctx, syncQueueKey).Result()
}

// RemoveStoryFromSyncViews removes story ID from sync queue after successful DB sync
func (c *RedisCache) RemoveStoryFromSyncViews(ctx context.Context, storyID string) error {
	rdb := c.client.GetClient()
	syncQueueKey := "stories:sync_views"
	return rdb.SRem(ctx, syncQueueKey, storyID).Err()
}

// -------------------------------------------------------------
// LIKE COUNT CACHING (HSET "likes:{content_type}:{content_id}")
// -------------------------------------------------------------

// AddLike caches a like in Redis Hash: likes:{content_type}:{content_id} -> {user_id} = "1"
func (c *RedisCache) AddLike(ctx context.Context, contentType string, contentID string, userID string) error {
	rdb := c.client.GetClient()
	likeKey := fmt.Sprintf("likes:%s:%s", contentType, contentID)

	// Set user like atomically
	if err := rdb.HSet(ctx, likeKey, userID, "1").Err(); err != nil {
		return fmt.Errorf("failed to hset like: %w", err)
	}

	// Queue this content for batch sync
	syncQueueKey := "likes:sync_queue"
	syncVal := fmt.Sprintf("%s:%s", contentType, contentID)
	if err := rdb.SAdd(ctx, syncQueueKey, syncVal).Err(); err != nil {
		c.logger.Warn("Failed to queue like sync", zap.Error(err), zap.String("sync_val", syncVal))
	}

	return nil
}

// RemoveLike removes a like cache
func (c *RedisCache) RemoveLike(ctx context.Context, contentType string, contentID string, userID string) error {
	rdb := c.client.GetClient()
	likeKey := fmt.Sprintf("likes:%s:%s", contentType, contentID)

	if err := rdb.HDel(ctx, likeKey, userID).Err(); err != nil {
		return fmt.Errorf("failed to hdel like: %w", err)
	}

	// Queue for batch sync
	syncQueueKey := "likes:sync_queue"
	syncVal := fmt.Sprintf("%s:%s", contentType, contentID)
	if err := rdb.SAdd(ctx, syncQueueKey, syncVal).Err(); err != nil {
		c.logger.Warn("Failed to queue like sync after removal", zap.Error(err))
	}

	return nil
}

// GetLikeCount retrieves the total likes count from Redis Hash length
func (c *RedisCache) GetLikeCount(ctx context.Context, contentType string, contentID string) (int64, error) {
	rdb := c.client.GetClient()
	likeKey := fmt.Sprintf("likes:%s:%s", contentType, contentID)

	count, err := rdb.HLen(ctx, likeKey).Result()
	if err != nil && err != goredis.Nil {
		return 0, err
	}
	return count, nil
}

// HasLiked checks if a user liked a content in Redis Cache
func (c *RedisCache) HasLiked(ctx context.Context, contentType string, contentID string, userID string) (bool, error) {
	rdb := c.client.GetClient()
	likeKey := fmt.Sprintf("likes:%s:%s", contentType, contentID)

	exists, err := rdb.HExists(ctx, likeKey, userID).Result()
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetLikesToSync retrieves all content entries requiring like count batch sync
func (c *RedisCache) GetLikesToSync(ctx context.Context) ([]string, error) {
	rdb := c.client.GetClient()
	syncQueueKey := "likes:sync_queue"
	return rdb.SMembers(ctx, syncQueueKey).Result()
}

// RemoveLikeFromSyncQueue removes content entry from like sync queue
func (c *RedisCache) RemoveLikeFromSyncQueue(ctx context.Context, syncVal string) error {
	rdb := c.client.GetClient()
	syncQueueKey := "likes:sync_queue"
	return rdb.SRem(ctx, syncQueueKey, syncVal).Err()
}

// GetLikeUserIDs retrieves all users who liked this content from Redis Hash
func (c *RedisCache) GetLikeUserIDs(ctx context.Context, contentType string, contentID string) ([]string, error) {
	rdb := c.client.GetClient()
	likeKey := fmt.Sprintf("likes:%s:%s", contentType, contentID)
	return rdb.HKeys(ctx, likeKey).Result()
}

// -------------------------------------------------------------
// COMMENT HOT SORT CACHING (Sorted Set 1-Minute Cache)
// -------------------------------------------------------------

// CacheCommentHotSort stores hot-sorted comments in a Sorted Set (ZSET) with 1 minute expiration
func (c *RedisCache) CacheCommentHotSort(ctx context.Context, contentID string, contentType string, comments []*domain.Comment) error {
	rdb := c.client.GetClient()
	key := fmt.Sprintf("comments:hot:%s:%s", contentType, contentID)

	// Clean existing key first
	rdb.Del(ctx, key)

	if len(comments) == 0 {
		return nil
	}

	zMembers := make([]goredis.Z, len(comments))
	for i, comment := range comments {
		// Hot Sort Score Formula: LikeCount + (Seconds since epoch / 45000)
		score := float64(comment.LikeCount) + float64(comment.CreatedAt.Unix())/45000.0
		
		commentBytes, err := json.Marshal(comment)
		if err != nil {
			return fmt.Errorf("failed to marshal comment: %w", err)
		}

		zMembers[i] = goredis.Z{
			Score:  score,
			Member: string(commentBytes),
		}
	}

	// Add to Redis ZSET
	if err := rdb.ZAdd(ctx, key, zMembers...).Err(); err != nil {
		return fmt.Errorf("failed to zadd comments: %w", err)
	}

	// Expire in 1 minute
	return rdb.Expire(ctx, key, 1*time.Minute).Err()
}

// GetCommentHotSort retrieves hot-sorted comments from cache
func (c *RedisCache) GetCommentHotSort(ctx context.Context, contentID string, contentType string) ([]*domain.Comment, error) {
	rdb := c.client.GetClient()
	key := fmt.Sprintf("comments:hot:%s:%s", contentType, contentID)

	// Fetch reverse range (highest score first)
	results, err := rdb.ZRevRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil // Cache miss or empty
	}

	comments := make([]*domain.Comment, len(results))
	for i, str := range results {
		comment := &domain.Comment{}
		if err := json.Unmarshal([]byte(str), comment); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cached comment: %w", err)
		}
		comments[i] = comment
	}

	return comments, nil
}

// -------------------------------------------------------------
// STORY EXPIRATION QUEUE (ZSET "stories:expiry")
// -------------------------------------------------------------

// AddStoryToExpiryQueue adds a story to the ZSET expiration queue
func (c *RedisCache) AddStoryToExpiryQueue(ctx context.Context, storyID string, expiresAt time.Time) error {
	rdb := c.client.GetClient()
	key := "stories:expiry"
	return rdb.ZAdd(ctx, key, goredis.Z{
		Score:  float64(expiresAt.Unix()),
		Member: storyID,
	}).Err()
}

// GetExpiredStories retrieves story IDs that are past their expiration time
func (c *RedisCache) GetExpiredStories(ctx context.Context, now time.Time) ([]string, error) {
	rdb := c.client.GetClient()
	key := "stories:expiry"
	return rdb.ZRangeByScore(ctx, key, &goredis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(now.Unix(), 10),
	}).Result()
}

// RemoveStoryFromExpiryQueue removes story ID from expiration queue
func (c *RedisCache) RemoveStoryFromExpiryQueue(ctx context.Context, storyID string) error {
	rdb := c.client.GetClient()
	key := "stories:expiry"
	return rdb.ZRem(ctx, key, storyID).Err()
}

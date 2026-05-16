package cache

import (
	"context"
	"encoding/json"
	"time"

	"nvide-live/pkg/redis"

	"go.uber.org/zap"
)

// Cache interface defines standard caching operations
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	InvalidatePattern(ctx context.Context, pattern string) error
}

type redisCache struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisCache creates a new Cache implementation using Redis
func NewRedisCache(client *redis.Client, logger *zap.Logger) Cache {
	return &redisCache{
		client: client,
		logger: logger,
	}
}

// Get retrieves a value from cache and unmarshals it into dest
func (c *redisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.GetClient().Get(ctx, key).Result()
	if err != nil {
		return err // returns redis.Nil if not found
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set marshals a value and stores it in cache
func (c *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.GetClient().Set(ctx, key, data, expiration).Err()
}

// Delete removes a key from cache
func (c *redisCache) Delete(ctx context.Context, key string) error {
	return c.client.GetClient().Del(ctx, key).Err()
}

// InvalidatePattern deletes all keys matching a pattern (use carefully)
func (c *redisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	iter := c.client.GetClient().Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		err := c.client.GetClient().Del(ctx, iter.Val()).Err()
		if err != nil {
			c.logger.Error("Failed to delete key during invalidation", zap.String("key", iter.Val()), zap.Error(err))
		}
	}
	return iter.Err()
}

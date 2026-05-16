package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client wraps Redis client with helper methods
type Client struct {
	rdb    *redis.Client
	logger *zap.Logger
}

// Config holds Redis configuration
type Config struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

// New creates a new Redis client
func New(cfg *Config, logger *zap.Logger) (*Client, error) {
	opt := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	}

	rdb := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	logger.Info("Redis connection established", zap.String("addr", cfg.Addr))

	return &Client{rdb: rdb, logger: logger}, nil
}

// Get retrieves value from Redis
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Set sets value in Redis with optional expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Del deletes key from Redis
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Exists checks if key exists
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.rdb.Exists(ctx, keys...).Result()
}

// Expire sets expiration on key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.rdb.Expire(ctx, key, expiration).Err()
}

// TTL gets remaining TTL for key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.rdb.TTL(ctx, key).Result()
}

// Close closes Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// IncrBy increments key value by amount
func (c *Client) IncrBy(ctx context.Context, key string, amount int64) (int64, error) {
	return c.rdb.IncrBy(ctx, key, amount).Result()
}

// Health checks Redis health
func (c *Client) Health(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Client) GetClient() *redis.Client {
	return c.rdb
}


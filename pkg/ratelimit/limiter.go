package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBucketLimiter implements a token bucket algorithm using Redis Lua scripts
type TokenBucketLimiter struct {
	client redis.Cmdable
	script *redis.Script
}

// Lua script for token bucket
const tokenBucketScript = `
local tokens_key = KEYS[1]
local timestamp_key = KEYS[2]

local rate = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local fill_time = capacity / rate
local ttl = math.floor(fill_time * 2)
if ttl < 60 then
    ttl = 60
end

local last_tokens = tonumber(redis.call("get", tokens_key))
if last_tokens == nil then
  last_tokens = capacity
end

local last_refreshed = tonumber(redis.call("get", timestamp_key))
if last_refreshed == nil then
  last_refreshed = 0
end

local delta = math.max(0, now - last_refreshed)
local filled_tokens = math.min(capacity, last_tokens + (delta * rate))
local allowed = filled_tokens >= requested

local new_tokens = filled_tokens
if allowed then
  new_tokens = filled_tokens - requested
end

redis.call("setex", tokens_key, ttl, new_tokens)
redis.call("setex", timestamp_key, ttl, now)

return { allowed and 1 or 0, new_tokens }
`

// NewTokenBucketLimiter creates a new TokenBucketLimiter
func NewTokenBucketLimiter(client redis.Cmdable) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		client: client,
		script: redis.NewScript(tokenBucketScript),
	}
}

// Allow checks if the request is allowed.
// rate: tokens added per second
// capacity: maximum token bucket size
// tokens: number of tokens requested
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string, rate float64, capacity float64, tokens int) (bool, float64, error) {
	tokensKey := fmt.Sprintf("ratelimit:tb:%s:tokens", key)
	tsKey := fmt.Sprintf("ratelimit:tb:%s:ts", key)

	now := float64(time.Now().UnixNano()) / 1e9 // timestamp in seconds with fractional part

	keys := []string{tokensKey, tsKey}
	args := []interface{}{rate, capacity, now, tokens}

	result, err := l.script.Run(ctx, l.client, keys, args...).Result()
	if err != nil {
		return false, 0, err
	}

	resArr, ok := result.([]interface{})
	if !ok || len(resArr) != 2 {
		return false, 0, fmt.Errorf("unexpected script result format")
	}

	allowedInt, ok := resArr[0].(int64)
	if !ok {
		return false, 0, fmt.Errorf("unexpected allowed format")
	}

	var remaining float64
	switch v := resArr[1].(type) {
	case int64:
		remaining = float64(v)
	case float64:
		remaining = v
	default:
		return false, 0, fmt.Errorf("unexpected remaining tokens format")
	}

	return allowedInt == 1, remaining, nil
}

package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTokenBucketLimiter(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	
	limiter := NewTokenBucketLimiter(rdb)
	ctx := context.Background()
	key := "test_key"
	
	// Test allowing requests within capacity
	// Rate: 1 token/sec, Capacity: 5 tokens
	for i := 0; i < 5; i++ {
		allowed, rem, err := limiter.Allow(ctx, key, 1, 5, 1)
		if err != nil {
			t.Errorf("Allow error: %v", err)
		}
		if !allowed {
			t.Errorf("Expected allowed=true for request %d", i+1)
		}
		if rem != float64(5-i-1) {
			t.Errorf("Expected remaining=%f, got %f", float64(5-i-1), rem)
		}
	}
	
	// Next request should be blocked
	allowed, _, err := limiter.Allow(ctx, key, 1, 5, 1)
	if err != nil {
		t.Errorf("Allow error: %v", err)
	}
	if allowed {
		t.Error("Expected allowed=false when capacity exceeded")
	}
	
	// Fast-forward time
	s.FastForward(2 * time.Second)
	
	// Should allow 2 more requests
	allowed, _, _ = limiter.Allow(ctx, key, 1, 5, 1)
	if !allowed {
		t.Error("Expected allowed=true after time refill")
	}
}

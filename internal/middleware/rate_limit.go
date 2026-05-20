package middleware

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"nvide-live/pkg/ratelimit"
	"nvide-live/pkg/redis"
)

// RateLimitMiddleware handles rate limiting using Redis
type RateLimitMiddleware struct {
	limiter *ratelimit.TokenBucketLimiter
	logger  *zap.Logger
	enabled bool
	rate    float64 // tokens per second
	capacity float64 // max tokens
}

// NewRateLimitMiddleware creates new rate limit middleware using token bucket
func NewRateLimitMiddleware(redisClient *redis.Client, logger *zap.Logger, enabled bool, requests int, window time.Duration) *RateLimitMiddleware {
	rate := float64(requests) / window.Seconds()
	return &RateLimitMiddleware{
		limiter:  ratelimit.NewTokenBucketLimiter(redisClient.GetClient()),
		logger:   logger,
		enabled:  enabled,
		rate:     rate,
		capacity: float64(requests),
	}
}

// isExemptedPath returns true if the path should skip rate limiting.
func isExemptedPath(path string) bool {
	switch path {
	case "/health", "/ready", "/metrics":
		return true
	}
	return false
}

// Middleware implements rate limiting using token bucket
func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		if isExemptedPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := getClientIP(r)
		key := "api:ip:" + clientIP

		allowed, remaining, err := m.limiter.Allow(r.Context(), key, m.rate, m.capacity, 1)
		if err != nil {
			m.logger.Error("Rate limiter error", zap.Error(err), zap.String("ip", clientIP))
			next.ServeHTTP(w, r)
			return
		}

		if !allowed {
			w.Header().Set("Retry-After", "1") // generic retry after 1s for token bucket
			WriteJSONError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests")
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", m.capacity))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", remaining))

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	ip := r.RemoteAddr
	for i := 0; i < len(ip); i++ {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}

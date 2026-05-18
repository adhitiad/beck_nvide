package wallet

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"nvide-live/pkg/redis"
)

// IdempotencyManager handles request idempotency using Redis or in-memory fallback
type IdempotencyManager struct {
	redisClient *redis.Client
	logger      *zap.Logger
	mu          sync.Mutex
	inMemory    map[string]time.Time
	isFallback  bool
}

// NewIdempotencyManager creates a new IdempotencyManager
func NewIdempotencyManager(redisClient *redis.Client, logger *zap.Logger) *IdempotencyManager {
	manager := &IdempotencyManager{
		redisClient: redisClient,
		logger:      logger,
		inMemory:    make(map[string]time.Time),
	}

	// Verify Redis health
	if redisClient == nil || redisClient.Health(context.Background()) != nil {
		manager.isFallback = true
		logger.Warn("Redis is unavailable. Idempotency manager falling back to in-memory mode.")
	}

	// Start in-memory cleanup routine
	go manager.cleanupLoop()

	return manager
}

// AcquireKey checks if the key exists. If not, it reserves it for 24 hours (86400 seconds).
// Returns true if the key is acquired (unique), false if it already exists (duplicate).
func (im *IdempotencyManager) AcquireKey(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, nil
	}

	redisKey := "idempotency:" + key

	// Try using Redis if not in fallback mode
	if !im.isFallback && im.redisClient != nil && im.redisClient.Health(ctx) == nil {
		// SET key "processing" NX EX 86400
		res, err := im.redisClient.GetClient().SetNX(ctx, redisKey, "locked", 24*time.Hour).Result()
		if err == nil {
			return res, nil
		}
		im.logger.Warn("Redis error during idempotency check, dynamically falling back to in-memory", zap.Error(err))
	}

	// Fallback to in-memory
	im.mu.Lock()
	defer im.mu.Unlock()

	if expireTime, exists := im.inMemory[key]; exists {
		if time.Now().Before(expireTime) {
			return false, nil // Duplicate
		}
	}

	// Set key to expire in 24 hours
	im.inMemory[key] = time.Now().Add(24 * time.Hour)
	return true, nil
}

func (im *IdempotencyManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		im.mu.Lock()
		now := time.Now()
		for k, expireTime := range im.inMemory {
			if now.After(expireTime) {
				delete(im.inMemory, k)
			}
		}
		im.mu.Unlock()
	}
}

// IsFinancialPath checks if the HTTP path is a financial endpoint that requires idempotency
func IsFinancialPath(path string) bool {
	financialSuffixes := []string{
		"/wallet/balance",       // GET, typically safe but we filter for POST below
		"/withdrawals",          // POST
		"/gifts/send",           // POST
		"/payment/deposit",      // POST
		"/payment/withdraw",     // POST
		"/crypto/withdrawal",    // POST
		"/conversations/",       // POST conversation unlocks/actions
		"/bookings",             // POST bookings
		"/offers/",              // POST bookings
	}

	for _, suffix := range financialSuffixes {
		if strings.Contains(path, suffix) {
			return true
		}
	}
	return false
}

// Middleware creates an HTTP middleware that enforces X-Idempotency-Key on financial POST requests
func (im *IdempotencyManager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Enforce idempotency ONLY on POST requests to financial endpoints
			if r.Method == http.MethodPost && IsFinancialPath(r.URL.Path) {
				idempotencyKey := r.Header.Get("X-Idempotency-Key")
				if idempotencyKey == "" {
					im.logger.Warn("Financial POST request missing X-Idempotency-Key", zap.String("path", r.URL.Path))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":   "MISSING_IDEMPOTENCY_KEY",
						"message": "X-Idempotency-Key header wajib disertakan untuk semua transaksi finansial.",
					})
					return
				}

				// Check or set the key
				acquired, err := im.AcquireKey(r.Context(), idempotencyKey)
				if err != nil {
					im.logger.Error("Idempotency check error", zap.Error(err))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":   "INTERNAL_ERROR",
						"message": "Gagal memproses validasi idempotensi.",
					})
					return
				}

				if !acquired {
					im.logger.Warn("Duplicate financial request detected", 
						zap.String("path", r.URL.Path), 
						zap.String("idempotency_key", idempotencyKey),
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict) // 409 Conflict
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error":   "CONFLICT",
						"message": "Transaksi dengan kunci idempotensi ini sedang atau telah diproses.",
					})
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

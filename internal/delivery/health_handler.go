package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"nvide-live/pkg/metrics"
	"nvide-live/pkg/redis"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db     *pgxpool.Pool
	redis  *redis.Client
	broker interface {
		Publish(ctx context.Context, topic string, msg string) error
	}
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client, broker interface{ Publish(ctx context.Context, topic string, msg string) error }) *HealthHandler {
	return &HealthHandler{db: db, redis: redis, broker: broker}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 1. Database Ping + Stats
	dbStatus := "ok"
	dbStats := map[string]interface{}{}
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "error: " + err.Error()
	} else {
		stats := h.db.Stat()
		dbStats["total_conns"] = stats.TotalConns()
		dbStats["idle_conns"] = stats.IdleConns()
		dbStats["acquired_conns"] = stats.AcquiredConns()
		dbStats["max_conns"] = stats.MaxConns()
	}

	// 2. Redis Ping + Latency
	redisStatus := "ok"
	var redisLatencyMs int64
	if h.redis != nil {
		start := time.Now()
		if err := h.redis.GetClient().Ping(ctx).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		} else {
			redisLatencyMs = time.Since(start).Milliseconds()
		}
	} else {
		redisStatus = "not_configured"
	}

	// 3. Broker Health check via Publish to a dry-run test topic
	brokerStatus := "ok"
	if h.broker != nil {
		// Attempt a brief publish to check Broker health (Redis pub/sub or mock)
		if err := h.broker.Publish(ctx, "health_check_dryrun", "ping"); err != nil {
			brokerStatus = "error: " + err.Error()
		}
	} else {
		brokerStatus = "not_configured"
	}

	// 4. Goroutines & Memory Stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	goroutinesActive := runtime.NumGoroutine()

	resp := map[string]interface{}{
		"status": "ok",
		"checks": map[string]interface{}{
			"database": map[string]interface{}{
				"status": dbStatus,
				"stats":  dbStats,
			},
			"redis": map[string]interface{}{
				"status":     redisStatus,
				"latency_ms": redisLatencyMs,
			},
			"broker": map[string]interface{}{
				"status": brokerStatus,
			},
		},
		"system": map[string]interface{}{
			"goroutines_active": goroutinesActive,
			"heap_alloc_mb":     float64(memStats.HeapAlloc) / 1024 / 1024,
			"sys_memory_mb":     float64(memStats.Sys) / 1024 / 1024,
		},
		"metrics": metrics.GetDefault().GetSnapshot(),
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "ok" || (redisStatus != "ok" && redisStatus != "not_configured") || (brokerStatus != "ok" && brokerStatus != "not_configured") {
		w.WriteHeader(http.StatusServiceUnavailable)
		resp["status"] = "error"
	} else {
		w.WriteHeader(http.StatusOK)
	}

	_ = json.NewEncoder(w).Encode(resp)
}

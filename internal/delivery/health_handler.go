package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"nvide-live/pkg/metrics"
	"nvide-live/pkg/redis"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	db     *pgxpool.Pool
	redis  *redis.Client
	broker interface {
		// Simple check interface
		Publish(ctx context.Context, topic string, msg string) error
	}
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client, broker interface{ Publish(ctx context.Context, topic string, msg string) error }) *HealthHandler {
	return &HealthHandler{db: db, redis: redis, broker: broker}
}

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "error: " + err.Error()
	}

	redisStatus := "ok"
	if h.redis != nil {
		if err := h.redis.GetClient().Ping(ctx).Err(); err != nil {
			redisStatus = "error: " + err.Error()
		}
	}

	resp := map[string]interface{}{
		"status": "ok",
		"checks": map[string]string{
			"database": dbStatus,
			"redis":    redisStatus,
		},
		"metrics": metrics.GetDefault().GetSnapshot(),
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "ok" || redisStatus != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
		resp["status"] = "error"
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(resp)
}

package middleware

import (
	"net/http"
	"strconv"
	"time"

	"nvide-live/pkg/metrics"
)

// responseWriterDelegator wraps a standard ResponseWriter to record status code
type responseWriterDelegator struct {
	http.ResponseWriter
	status int
}

func (rd *responseWriterDelegator) WriteHeader(status int) {
	rd.status = status
	rd.ResponseWriter.WriteHeader(status)
}

func (rd *responseWriterDelegator) Write(b []byte) (int, error) {
	if rd.status == 0 {
		rd.status = http.StatusOK
	}
	return rd.ResponseWriter.Write(b)
}

// MetricsMiddleware tracks HTTP request count, duration, and status codes for Prometheus
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ignore /metrics and /health from request metrics to avoid noise
		if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rd := &responseWriterDelegator{ResponseWriter: w}

		next.ServeHTTP(rd, r)

		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(rd.status)
		if rd.status == 0 {
			statusStr = "200"
		}

		m := metrics.GetDefault()
		m.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, statusStr).Inc()
		m.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

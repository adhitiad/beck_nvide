package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type loggingContextKey string

const CorrelationIDKey loggingContextKey = "correlation_id"

// LoggingMiddleware logs HTTP requests with correlation ID
func LoggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Get or create correlation ID
			correlationID := r.Header.Get("X-Correlation-ID")
			if correlationID == "" {
				correlationID = uuid.New().String()
			}

			// Inject correlation ID into context and response header
			ctx := context.WithValue(r.Context(), CorrelationIDKey, correlationID)
			w.Header().Set("X-Correlation-ID", correlationID)

			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r.WithContext(ctx))

			duration := time.Since(start)
			
			// Extract user_id from context if available (set by AuthMiddleware)
			userID := ""
			if uid := r.Context().Value(userIDKey); uid != nil {
				userID = uid.(string)
			}

			logger.Info("HTTP request",
				zap.String("correlation_id", correlationID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.statusCode),
				zap.Duration("duration", duration),
				zap.String("user_id", userID),
				zap.String("user_agent", r.UserAgent()),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// CORSMiddleware is a mux-compatible middleware (kept for compatibility)
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SetCORSHeaders(w)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSWrapper wraps an http.Handler to provide global CORS support.
// Use this instead of CORSMiddleware for the main server handler.
func CORSWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetCORSHeaders(w)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// SetCORSHeaders sets standard CORS headers
func SetCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Correlation-ID")
}


// RecoveryMiddleware recovers from panics
func RecoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					correlationID, _ := r.Context().Value(CorrelationIDKey).(string)
					logger.Error("Panic recovered",
						zap.String("correlation_id", correlationID),
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
					)
					writeJSONErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func writeJSONErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error_code": code,
		"message":    message,
	})
}

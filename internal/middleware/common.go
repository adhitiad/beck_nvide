package middleware

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
)

type loggingContextKey string

const CorrelationIDKey loggingContextKey = "correlation_id"


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



func writeJSONErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error_code": code,
		"message":    message,
	})
}

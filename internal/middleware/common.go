package middleware

import (
	"encoding/json"
	"go.uber.org/zap"
	"net/http"
)

type loggingContextKey string

const CorrelationIDKey loggingContextKey = "correlation_id"

// RedactURL strips credentials from a database URL for safe log output.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}
	// Handles postgres://user:pass@host:5432/dbname
	prefixEnd := 0
	for i, c := range raw {
		if c == ':' {
			prefixEnd = i
			break
		}
	}
	if prefixEnd == 0 {
		return raw
	}
	scheme := raw[:prefixEnd]
	rest := raw[prefixEnd:]
	for i, c := range rest {
		if c == '/' && i+2 < len(rest) && rest[i+1] == '/' {
			credStart := i + 2
			atIdx := -1
			for j := credStart; j < len(rest); j++ {
				if rest[j] == '@' {
					atIdx = j
					break
				}
			}
			if atIdx > credStart {
				redacted := make([]byte, len(rest))
				copy(redacted, rest[:credStart])
				for k := credStart; k < atIdx; k++ {
					redacted[k] = '*'
				}
				copy(redacted[atIdx:], rest[atIdx:])
				return scheme + string(redacted)
			}
		}
	}
	return raw
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
					WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// Standardised JSON response envelope
// ---------------------------------------------------------------------------

// Envelope is the standard JSON response shape used by every API handler.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError is the structured error shape inside the envelope.
type APIError struct {
	Code    string `json:"error_code"`
	Message string `json:"message"`
}

// WriteJSON writes a standardised JSON envelope response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

// WriteJSONError writes a standardised JSON error envelope response.
func WriteJSONError(w http.ResponseWriter, status int, code, message string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

// MaxRequestBodyBytes is the global default for request body size limiting (10 MB).
const MaxRequestBodyBytes = 10 << 20

// BodyLimitMiddleware wraps r.Body with http.MaxBytesReader to prevent large payload attacks.
func BodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

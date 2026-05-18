package middleware

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"

	"nvide-live/pkg/logger"
	"nvide-live/pkg/uuid"
)

const RequestIDCtxKey contextKey = "request_id"

// RequestID middleware inject request ID ke context menggunakan UUID v7
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = r.Header.Get("X-Correlation-ID")
		}
		if reqID == "" {
			reqID = uuid.NewV7()
		}

		ctx := context.WithValue(r.Context(), logger.RequestIDKey, reqID)
		ctx = context.WithValue(ctx, RequestIDCtxKey, reqID)

		w.Header().Set("X-Request-ID", reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggerMiddleware HTTP request logging dengan correlation + Hijacker support untuk WebSockets
func LoggerMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get requestID from context (set by RequestID middleware or generate one)
			ctx := r.Context()
			reqID, ok := ctx.Value(logger.RequestIDKey).(string)
			if !ok || reqID == "" {
				reqID = r.Header.Get("X-Request-ID")
				if reqID == "" {
					reqID = uuid.NewV7()
				}
				ctx = context.WithValue(ctx, logger.RequestIDKey, reqID)
				ctx = context.WithValue(ctx, RequestIDCtxKey, reqID)
				w.Header().Set("X-Request-ID", reqID)
			}

			// Wrap response writer untuk capture status & bytes
			ww := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				bytesWritten:   0,
			}

			// Extract user ID dari JWT claims jika ada
			if claims, ok := ctx.Value("user_claims").(map[string]interface{}); ok {
				if userID, ok := claims["user_id"].(string); ok {
					ctx = context.WithValue(ctx, logger.UserIDKey, userID)
				}
			}

			next.ServeHTTP(ww, r.WithContext(ctx))

			// Build log fields
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.Int("status", ww.statusCode),
				zap.Int("bytes", ww.bytesWritten),
				zap.Duration("duration_ms", time.Since(start)),
				zap.String("user_agent", r.UserAgent()),
				zap.String("ip", r.RemoteAddr),
				zap.String("referer", r.Referer()),
			}

			// Log level berdasarkan status
			entry := logger.WithContext(ctx)
			switch {
			case ww.statusCode >= 500:
				entry.Error("http_request_error", fields...)
			case ww.statusCode >= 400:
				entry.Warn("http_request_warning", fields...)
			case time.Since(start) > 500*time.Millisecond:
				entry.Warn("http_request_slow", fields...)
			default:
				entry.Info("http_request", fields...)
			}
		})
	}
}

// responseWriter wrapper untuk capture metrics, mendukung Hijacker untuk WebSockets
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	written      bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter does not implement http.Hijacker")
	}
	return hijacker.Hijack()
}

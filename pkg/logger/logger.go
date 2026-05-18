package logger

import (
	"context"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Context key untuk request correlation
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	SessionIDKey contextKey = "session_id"
)

var (
	globalLogger *zap.Logger
)

// InitLogger inisialisasi global logger dengan production config
func InitLogger(env string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Custom fields untuk semua log
	config.InitialFields = map[string]interface{}{
		"service": "nvide-api",
		"version": "1.0.0",
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// GetGlobal return global logger instance
func GetGlobal() *zap.Logger {
	if globalLogger == nil {
		// Fallback ke default
		l, _ := zap.NewProduction()
		globalLogger = l
	}
	return globalLogger
}

// WithContext membuat logger baru dengan context fields
func WithContext(ctx context.Context) *zap.Logger {
	logger := GetGlobal()

	// Extract correlation IDs dari context
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		logger = logger.With(zap.String("request_id", reqID))
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok && userID != "" {
		logger = logger.With(zap.String("user_id", userID))
	}
	if sessionID, ok := ctx.Value(SessionIDKey).(string); ok && sessionID != "" {
		logger = logger.With(zap.String("session_id", sessionID))
	}

	return logger
}

// WithFields tambah custom fields ke logger
func WithFields(fields ...zap.Field) *zap.Logger {
	return GetGlobal().With(fields...)
}

// Helper functions dengan context

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Debug(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Info(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Warn(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Error(msg, fields...)
}

func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	WithContext(ctx).Fatal(msg, fields...)
}

// Sync flush buffer sebelum shutdown
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Gin-style atau HTTP middleware helper

// HTTPLogEntry struktur log untuk HTTP request
type HTTPLogEntry struct {
	Logger    *zap.Logger
	StartTime time.Time
}

// NewHTTPLogEntry buat entry baru untuk request
func NewHTTPLogEntry(ctx context.Context) *HTTPLogEntry {
	return &HTTPLogEntry{
		Logger:    WithContext(ctx),
		StartTime: time.Now(),
	}
}

// LogHTTPRequest log complete HTTP request
func (e *HTTPLogEntry) LogHTTPRequest(
	method string,
	path string,
	status int,
	bytes int,
	userAgent string,
	ip string,
	err error,
) {
	duration := time.Since(e.StartTime)

	fields := []zap.Field{
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", status),
		zap.Int("bytes", bytes),
		zap.Duration("duration_ms", duration),
		zap.String("user_agent", userAgent),
		zap.String("ip", ip),
	}

	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	// Log level berdasarkan status
	switch {
	case status >= 500:
		e.Logger.Error("http_request_server_error", fields...)
	case status >= 400:
		e.Logger.Warn("http_request_client_error", fields...)
	case duration > 500*time.Millisecond:
		e.Logger.Warn("http_request_slow", fields...)
	default:
		e.Logger.Info("http_request", fields...)
	}
}

// Database query logger

// QueryLog log database query dengan context
func QueryLog(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) {
	logger := WithContext(ctx).With(
		zap.String("query", query),
		zap.Duration("duration_ms", duration),
	)

	if err != nil {
		logger.Error("database_query_error", zap.Error(err))
		return
	}

	if duration > 100*time.Millisecond {
		logger.Warn("database_query_slow")
		return
	}

	logger.Debug("database_query")
}

// Business event logger untuk audit trail

type AuditEvent struct {
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id,omitempty"`
	Resource  string                 `json:"resource"`
	Action    string                 `json:"action"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func AuditLog(ctx context.Context, event AuditEvent) {
	event.Timestamp = time.Now()

	logger := WithContext(ctx).With(
		zap.String("audit_event_type", event.EventType),
		zap.String("audit_resource", event.Resource),
		zap.String("audit_action", event.Action),
		zap.Any("audit_details", event.Details),
	)

	logger.Info("audit_log")
}

// Performance metrics helper

func PerformanceLog(ctx context.Context, operation string, duration time.Duration, extraFields ...zap.Field) {
	logger := WithContext(ctx).With(
		zap.String("operation", operation),
		zap.Duration("duration_ms", duration),
	)

	if duration > 1*time.Second {
		logger.Warn("operation_slow", extraFields...)
	} else {
		logger.Debug("operation", extraFields...)
	}
}

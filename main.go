package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"nvide-live/cmd/server"
	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
	"nvide-live/internal/middleware"
	workerV1 "nvide-live/internal/worker"
	"nvide-live/pkg/config"
	"nvide-live/pkg/i18n"
	pkgLogger "nvide-live/pkg/logger"
	"nvide-live/pkg/rbac"
	"nvide-live/pkg/worker"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found, using system environment variables\n")
	}

	// Load configuration
	cfg := config.Load()

	// Setup logger
	logger := setupLogger(cfg.LogLevel, cfg.LogFormat)
	defer logger.Sync()

	// Initialize global pkg/logger package (Problem 6)
	if err := pkgLogger.InitLogger("production"); err != nil {
		fmt.Printf("Warning: failed to initialize pkg/logger: %v\n", err)
	}
	defer pkgLogger.Sync()

	// Initialize Multi-Language translations (i18n)
	i18nTranslator := i18n.GetTranslator()
	if err := i18nTranslator.LoadTranslations("pkg/i18n/locales"); err != nil {
		logger.Warn("Failed to load translations from disk, using embedded fallback", zap.Error(err))
	} else {
		logger.Info("Multi-language translations (i18n) loaded successfully")
	}

	logger.Info("Starting NVide Live Platform - Fase 5: Scaling & Optimization",
		zap.String("version", "1.1.0"),
		zap.String("env", "production"),
	)

	// Initialize App using Wire
	app, err := server.InitializeApp(logger)
	if err != nil {
		logger.Fatal("Failed to initialize application components", zap.Error(err))
	}
	defer app.DB.Close()
	defer app.Redis.Close()

	// Bypass auto-migration since tables already exist
	logger.Info("Database auto-migration skipped (tables already migrated)")

	// Initialize and start old background worker (legacy compatibility)
	bgWorker := workerV1.NewWorker(app.DB.Pool(), app.Redis, logger)
	bgWorker.Start()
	defer bgWorker.Stop()

	// Load RBAC Permissions
	loadRBACPermissions(context.Background(), app.RBACManager, app.RoleRepo, logger)

	// Initialize and start background ModerationWorker
	moderationWorker := workerV1.NewModerationWorker(app.ModerationUseCase, logger)
	go moderationWorker.Start(context.Background())
	defer moderationWorker.Stop()

	// Wait Room & Stream Schedule initialization
	go app.WaitRoomHub.Run()
	go app.WSHub.Run()

	app.WorkerPool.Start()
	defer app.WorkerPool.Stop()

	// Initialize new unified BackgroundJobManager
	jobManager := workerV1.NewBackgroundJobManager(logger)

	// LevelingUseCase (Job Handlers)
	app.WorkerPool.RegisterHandler(worker.JobXPBatchUpdate, func(ctx context.Context, job *worker.Job) error {
		return app.LevelingUseCase.FlushXPUpdates(ctx)
	})

	// 1. Periodic XP Batch Update Trigger
	jobManager.Register("xp-flush", 10*time.Second, func(ctx context.Context) {
		app.WorkerPool.Enqueue(&worker.Job{
			ID:        fmt.Sprintf("xp-flush-%d", time.Now().Unix()),
			Type:      worker.JobXPBatchUpdate,
			CreatedAt: time.Now(),
		})
	})

	// 2. Periodic Trending Score Recalculator
	jobManager.Register("trending-recalc", 30*time.Second, func(ctx context.Context) {
		_ = app.TrendingUseCase.RecalculateTrendingScores(ctx)
	})

	app.WorkerPool.RegisterHandler(worker.JobVideoTranscode, func(ctx context.Context, job *worker.Job) error {
		var payload usecase.VODTranscodePayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		vodID, err := domain.FromString(payload.VODID)
		if err != nil {
			return err
		}
		return app.VODUseCase.ProcessVideo(ctx, vodID, payload.TempFilePath, payload.OriginalFileName)
	})

	// 3. Periodic Smart Reminder Check
	jobManager.Register("smart-reminders", 1*time.Minute, func(ctx context.Context) {
		_ = app.ScheduleUseCase.CheckAndSendTieredReminders(ctx)
	})

	// 4. Daily Occurrence Generator Refill Job
	jobManager.RegisterWithDelay("occurrence-refill", 12*time.Hour, 5*time.Second, func(ctx context.Context) {
		_ = app.ScheduleUseCase.RefillAllOccurrences(ctx)
	})

	go app.CryptoMonitor.Start(context.Background())
	defer app.CryptoMonitor.Stop()

	// 5. Periodic Disappearing Messages Processor
	jobManager.Register("disappearing-msgs", 5*time.Second, func(ctx context.Context) {
		_ = app.PrivateChatUseCase.ProcessExpiredMessages(ctx)
	})

	// 6. Background Workers - VIP Expiry Check
	jobManager.Register("vip-expiry", 1*time.Hour, func(ctx context.Context) {
		_ = app.VIPUseCase.ProcessExpiredVIP(ctx)
	})

	// 7. Background Workers - Host Level Evaluation
	jobManager.Register("host-level-eval", 6*time.Hour, func(ctx context.Context) {
		_ = app.HostLevelUseCase.EvaluateAndPromote(ctx)
	})

	// WebRTC Room Manager (Internal sync)
	app.StreamUseCase.StartViewerCountSyncJob(context.Background(), app.WebRTCRoomManager)

	// Start all periodic background jobs
	jobManager.StartAll()
	defer jobManager.StopAll()

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort),
		Handler:      middleware.CORSWrapper(app.Router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting",
			zap.String("addr", server.Addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.GracefulTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// 1. Drain WebSocket connections gracefully (Fase 5)
	logger.Info("Stopping WebSocket Hub...")
	app.WSHub.Stop()

	// 2. Flush critical Redis caches or close cleanly
	if app.Redis != nil {
		logger.Info("Closing Redis client...")
		_ = app.Redis.Close()
	}

	// 3. Close database pools gracefully
	if app.DB != nil {
		logger.Info("Closing PostgreSQL connection pool...")
		app.DB.Close()
	}

	logger.Info("Server exited cleanly")
}

// setupLogger configures zap logger
func setupLogger(level, format string) *zap.Logger {
	var logLevel zapcore.Level
	switch level {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	default:
		logLevel = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		logLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return logger
}

// loadRBACPermissions loads all permissions into RBAC manager
func loadRBACPermissions(
	ctx context.Context,
	manager *rbac.Manager,
	roleRepo domain.RoleRepository,
	logger *zap.Logger,
) {
	// Get all roles
	roles, err := roleRepo.List(ctx)
	if err != nil {
		logger.Error("Failed to load roles for RBAC", zap.Error(err))
		return
	}

	// Load permissions for each role
	for _, role := range roles {
		permissions, err := roleRepo.GetPermissionsByRoleID(ctx, role.ID)
		if err != nil {
			logger.Error("Failed to load permissions for role",
				zap.Error(err),
				zap.String("role", role.Name),
			)
			continue
		}
		manager.LoadPermissions(role.Name, permissions)
		logger.Debug("Loaded permissions for role",
			zap.String("role", role.Name),
			zap.Int("permission_count", len(permissions)),
		)
	}

	logger.Info("RBAC permissions loaded",
		zap.Int("roles_loaded", len(roles)),
	)
}

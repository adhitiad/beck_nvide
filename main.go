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

	"nvide-live/internal/delivery"
	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
	"nvide-live/internal/repository"
	"nvide-live/internal/usecase"
	"nvide-live/internal/webrtc"
	"nvide-live/internal/websocket"
	workerV1 "nvide-live/internal/worker"
	"nvide-live/pkg/auth"
	"nvide-live/pkg/blockchain"
	"nvide-live/pkg/broker"
	"nvide-live/pkg/config"
	"nvide-live/pkg/database"
	"nvide-live/pkg/duitku"
	"nvide-live/pkg/ffmpeg"
	"nvide-live/pkg/i18n"
	"nvide-live/pkg/rbac"
	"nvide-live/pkg/redis"
	"nvide-live/pkg/storage"
	"nvide-live/pkg/wallet"
	"nvide-live/pkg/worker"
	pkgLogger "nvide-live/pkg/logger"
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

	// Initialize database
	db, err := database.New(&database.Config{
		DATABASE_URL: cfg.DATABASE_URL,
		Host:         cfg.DBHost,
		Port:         cfg.DBPort,
		User:         cfg.DBUser,
		Password:     cfg.DBPassword,
		DBName:       cfg.DBName,
		SSLMode:      cfg.DBSSLMode,
		MaxConn:      cfg.DBMaxConn,
		MinConn:      cfg.DBMinConn,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Auto-migrate database tables
	connStr := cfg.DATABASE_URL
	if connStr == "" {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)
	}
	// Bypass auto-migration since tables already exist and remote GORM checks can hang
	logger.Info("Database auto-migration skipped (tables already migrated)")


	// Initialize Redis
	redisClient, err := redis.New(&redis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: cfg.RedisPoolSize,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.Pool(), logger)
	tokenRepo := repository.NewTokenRepository(db.Pool(), logger)
	roleRepo := repository.NewRoleRepository(db.Pool(), logger)

	// Initialize auth service
	authService := auth.New(cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshTokenExpiry)

	// Initialize RBAC manager and load permissions
	rbacManager := rbac.New()
	loadRBACPermissions(context.Background(), rbacManager, roleRepo, logger)

	// Initialize usecases
	authUseCase := usecase.NewAuthUseCase(
		userRepo,
		tokenRepo,
		roleRepo,
		authService,
		redisClient,
		rbacManager,
		logger,
		cfg.JWTExpiry,
		cfg.RefreshTokenExpiry,
	)

	// Initialize social repositories
	storyRepo := repository.NewStoryRepository(db.Pool(), logger)
	storyViewRepo := repository.NewStoryViewRepository(db.Pool(), logger)
	commentRepo := repository.NewCommentRepository(db.Pool(), logger)
	commentLikeRepo := repository.NewCommentLikeRepository(db.Pool(), logger)
	likeRepo := repository.NewLikeRepository(db.Pool(), logger)
	messageRepo := repository.NewMessageRepository(db.Pool(), logger)
	chatRoomRepo := repository.NewChatRoomRepository(db.Pool(), logger)
	streamRepo := repository.NewStreamRepository(db.Pool(), logger)
	streamSessionRepo := repository.NewStreamSessionRepository(db.Pool(), logger)
	pkRepo := repository.NewPKBattleRepository(db.Pool(), logger)
	vodRepo := repository.NewVODMediaRepository(db.Pool(), logger)
	walletRepo := repository.NewWalletRepository(db.Pool(), logger)
	txRepo := repository.NewTransactionRepository(db.Pool(), logger)
	agencyRepo := repository.NewAgencyRepository(db.Pool(), logger)
	hostAppRepo := repository.NewHostApplicationRepository(db.Pool(), logger)
	giftRepo := repository.NewGiftRepository(db.Pool(), logger)
	giftTxRepo := repository.NewGiftTransactionRepository(db.Pool(), logger)
	duitkuPaymentRepo := repository.NewDuitkuPaymentRepository(db.Pool(), logger)
	cryptoRepo := repository.NewCryptoRepository(db.Pool(), logger)
	privateChatRepo := repository.NewPrivateChatRepository(db.Pool(), logger)
	paidInteractionRepo := repository.NewPaidInteractionRepository(db.Pool(), logger)
	withdrawalRepo := repository.NewWithdrawalRepository(db.Pool(), logger)
	bookingRepo := repository.NewBookingRepository(db.Pool(), logger)
	moderationRepo := repository.NewModerationRepository(db.Pool(), logger)
	monetizationRepo := repository.NewMonetizationRepository(db.Pool(), logger)
	creatorTokenRepo := repository.NewCreatorTokenRepository(db.Pool(), logger)
	predictionRepo := repository.NewPredictionRepository(db.Pool(), logger)
	drmRepo := repository.NewDRMRepository(db.Pool(), logger)
	recommendationRepo := repository.NewRecommendationRepository(db.Pool(), logger)
	clipRepo := repository.NewClipRepository(db.Pool(), logger)

	// New KYC and Subscription Repositories
	kycRepo := repository.NewKYCRepository(db.Pool(), logger)
	bannedRepo := repository.NewBannedUserRepository(db.Pool(), logger)
	onboardRepo := repository.NewOnboardingRepository(db.Pool(), logger)
	clipSubRepo := repository.NewClipSubscriptionRepository(db.Pool(), logger)

	// Initialize Storage and FFmpeg
	localStorage := storage.NewLocalStorage("./uploads", "/uploads", logger)
	ffmpegSvc := ffmpeg.New(logger)

	// Initialize Duitku Client
	duitkuClient := duitku.NewClient(&duitku.Config{
		MerchantCode: cfg.DuitkuMerchantCode,
		APIKey:       cfg.DuitkuAPIKey,
		BaseURL:      cfg.DuitkuBaseURL,
		CallbackURL:  cfg.DuitkuCallbackURL,
		ReturnURL:    cfg.DuitkuReturnURL,
	}, logger)

	// Initialize Message Broker and WS Hub
	msgBroker := broker.NewHybridBroker(redisClient, logger)
	wsHub := websocket.NewHub(db.Pool(), msgBroker, redisClient, logger)
	go wsHub.Run()

	// Initialize Blockchain Clients
	solanaClient := blockchain.NewSolanaClient(cfg.SolanaRPCURL)
	evmClient, _ := blockchain.NewEVMClient(cfg.USDTRPCURL) // BSC Testnet for USDT

	// Initialize new worker pool (Fase 5)
	workerPool := worker.NewPool(5, logger)
	workerPool.Start()
	defer workerPool.Stop()

	// Initialize Leveling XP usecase & Register handler on workerPool (Fase 5)
	levelingUseCase := usecase.NewLevelingUseCase(db.Pool(), redisClient, wsHub, logger)
	workerPool.RegisterHandler(worker.JobXPBatchUpdate, func(ctx context.Context, job *worker.Job) error {
		return levelingUseCase.FlushXPUpdates(ctx)
	})

	// Periodic XP Batch Update Trigger (Fase 5)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			workerPool.Enqueue(&worker.Job{
				ID:        fmt.Sprintf("xp-flush-%d", time.Now().Unix()),
				Type:      worker.JobXPBatchUpdate,
				CreatedAt: time.Now(),
			})
		}
	}()

	// Initialize and start old background worker (legacy compatibility)
	bgWorker := workerV1.NewWorker(db.Pool(), redisClient, logger)
	bgWorker.Start()
	defer bgWorker.Stop()

	// Initialize user usecase (Fase 5: Cache & Singleflight enabled)
	userUseCase := usecase.NewUserUseCase(userRepo, redisClient, logger)

	// Initialize social usecases
	storyUseCase := usecase.NewStoryUseCase(storyRepo, storyViewRepo, userRepo, logger)
	commentUseCase := usecase.NewCommentUseCase(commentRepo, commentLikeRepo, userRepo, logger)
	likeUseCase := usecase.NewLikeUseCase(likeRepo, redisClient, msgBroker, logger)
	messageUseCase := usecase.NewMessageUseCase(messageRepo, chatRoomRepo, userRepo, logger)
	streamUseCase := usecase.NewStreamUseCase(streamRepo, streamSessionRepo, walletRepo, redisClient, msgBroker, logger)
	pkUseCase := usecase.NewPKBattleUseCase(pkRepo, streamRepo, wsHub, redisClient, logger)
	trendingUseCase := usecase.NewTrendingUseCase(db.Pool(), redisClient, streamRepo, logger)

	// Periodic Trending Score Recalculator (Fase 6)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = trendingUseCase.RecalculateTrendingScores(context.Background())
		}
	}()
	vodUseCase := usecase.NewVODUseCase(vodRepo, drmRepo, ffmpegSvc, localStorage, redisClient, workerPool, logger)
	workerPool.RegisterHandler(worker.JobVideoTranscode, func(ctx context.Context, job *worker.Job) error {
		var payload usecase.VODTranscodePayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return err
		}
		vodID, err := domain.FromString(payload.VODID)
		if err != nil {
			return err
		}
		return vodUseCase.ProcessVideo(ctx, vodID, payload.TempFilePath, payload.OriginalFileName)
	})
	walletUseCase := usecase.NewWalletUseCase(walletRepo, txRepo, redisClient, logger)
	privateChatUseCase := usecase.NewPrivateChatUsecase(privateChatRepo, userRepo, redisClient, logger)
	giftUseCase := usecase.NewGiftUseCase(giftRepo, giftTxRepo, agencyRepo, walletUseCase, privateChatUseCase, messageRepo, chatRoomRepo, wsHub, redisClient, logger)
	agencyUseCase := usecase.NewAgencyUseCase(hostAppRepo, agencyRepo, walletUseCase, logger)
	paymentUseCase := usecase.NewPaymentUseCase(txRepo, duitkuPaymentRepo, walletUseCase, duitkuClient, redisClient, logger)
	cryptoUseCase := usecase.NewCryptoUseCase(cryptoRepo, walletUseCase, redisClient, logger, []byte(cfg.CryptoEncryptionKey))
	paidInteractionUseCase := usecase.NewPaidInteractionUsecase(paidInteractionRepo, walletRepo, txRepo, userRepo, agencyRepo, redisClient, logger)
	withdrawalUseCase := usecase.NewWithdrawalUsecase(withdrawalRepo, walletRepo, txRepo, agencyRepo, userRepo, redisClient, logger)
	bookingUseCase := usecase.NewBookingUsecase(bookingRepo, walletRepo, agencyRepo, withdrawalUseCase, redisClient, logger)
	offerRepo := repository.NewOfferRepository(db.Pool(), logger)
	offerUseCase := usecase.NewOfferUsecase(offerRepo, bookingRepo, bookingUseCase, walletRepo, agencyRepo, redisClient, logger)
	locationUseCase := usecase.NewLocationUsecase(bookingRepo, redisClient, logger)
	creatorTokenUseCase := usecase.NewCreatorTokenUseCase(creatorTokenRepo, walletRepo, txRepo, logger)
	predictionUseCase := usecase.NewPredictionUseCase(predictionRepo, streamRepo, walletRepo, txRepo, creatorTokenRepo, logger)
	drmUseCase := usecase.NewDRMUseCase(drmRepo, vodRepo, logger)
	recommendationUseCase := usecase.NewRecommendationUseCase(recommendationRepo, streamRepo, vodRepo, logger)
	clipUseCase := usecase.NewClipUseCase(clipRepo, streamRepo, redisClient, logger)
	extraMonetizationUseCase := usecase.NewMonetizationUseCase(monetizationRepo, walletRepo, txRepo, streamRepo, logger)

	// New KYC and Subscription Usecases
	kycUseCase := usecase.NewKYCUseCase(kycRepo, userRepo, bannedRepo, streamRepo, onboardRepo, logger)
	onboardingUseCase := usecase.NewOnboardingUseCase(onboardRepo, userRepo, logger)
	clipSubUseCase := usecase.NewClipSubscriptionUseCase(clipSubRepo, userRepo, walletUseCase, logger)

	// Initialize ModerationUseCase & NSFW Scanner (Fitur 6)
	nsfwScanner := usecase.NewAWSRekognitionScanner(logger)
	moderationUseCase := usecase.NewModerationUseCase(moderationRepo, wsHub, redisClient, logger, nsfwScanner)
	wsHub.SetModerationUseCase(moderationUseCase)

	// Initialize and start background ModerationWorker
	moderationWorker := workerV1.NewModerationWorker(moderationUseCase, logger)
	go moderationWorker.Start(context.Background())
	defer moderationWorker.Stop()

	// Wait Room & Stream Schedule initialization (Fitur 7)
	waitRoomHub := websocket.NewWaitRoomHub(db.Pool(), redisClient, logger)
	go waitRoomHub.Run()

	scheduleRepo := repository.NewLiveScheduleRepository(db.Pool(), logger)
	scheduleUseCase := usecase.NewLiveScheduleUseCase(scheduleRepo, waitRoomHub, wsHub, redisClient, logger)

	// Periodic Smart Reminder Check (Fitur 7)
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			_ = scheduleUseCase.CheckAndSendTieredReminders(context.Background())
		}
	}()

	// Daily Occurrence Generator Refill Job (Fitur 7)
	go func() {
		time.Sleep(5 * time.Second) // Let system warm up first
		_ = scheduleUseCase.RefillAllOccurrences(context.Background())
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			_ = scheduleUseCase.RefillAllOccurrences(context.Background())
		}
	}()

	cryptoMonitor := workerV1.NewCryptoMonitor(cryptoRepo, cryptoUseCase, solanaClient, evmClient, logger)
	go cryptoMonitor.Start()
	defer cryptoMonitor.Stop()

	// Periodic Disappearing Messages Processor (Fitur 8)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = privateChatUseCase.ProcessExpiredMessages(context.Background())
		}
	}()

	// WebRTC Room Manager
	webrtcRoomManager := webrtc.NewRoomManager(logger)
	streamUseCase.StartViewerCountSyncJob(context.Background(), webrtcRoomManager)

	// Initialize handlers
	handler := delivery.NewHandler(
		authUseCase,
		userUseCase,
		storyUseCase,
		commentUseCase,
		likeUseCase,
		messageUseCase,
		privateChatUseCase,
		paidInteractionUseCase,
		bookingUseCase,
		offerUseCase,
		locationUseCase,
		scheduleUseCase,
		waitRoomHub,
		wsHub,
		logger,
	)
	webrtcHandler := delivery.NewWebRTCHandler(webrtcRoomManager, streamUseCase, authUseCase, trendingUseCase, scheduleUseCase, logger)
	pkHandler := delivery.NewPKBattleHandler(pkUseCase, logger)
	vodHandler := delivery.NewVODHandler(vodUseCase, logger)
	monetizationHandler := delivery.NewMonetizationHandler(walletUseCase, giftUseCase, agencyUseCase, paymentUseCase, withdrawalUseCase, logger)
	extraMonetizationHandler := delivery.NewExtraMonetizationHandler(extraMonetizationUseCase, logger)
	healthHandler := delivery.NewHealthHandler(db.Pool(), redisClient, msgBroker)
	cryptoHandler := delivery.NewCryptoHandler(cryptoUseCase, cryptoMonitor, logger)
	moderationHandler := delivery.NewModerationHandler(moderationUseCase, logger)
	creatorTokenHandler := delivery.NewCreatorTokenHandler(creatorTokenUseCase, logger)
	predictionHandler := delivery.NewPredictionHandler(predictionUseCase, logger)
	drmHandler := delivery.NewDRMHandler(drmUseCase, vodUseCase, logger)
	recommendationHandler := delivery.NewRecommendationHandler(recommendationUseCase, logger)
	clipHandler := delivery.NewClipHandler(clipUseCase, logger)

	// New KYC and Subscription Handlers
	kycHandler := delivery.NewKYCHandler(kycUseCase, onboardingUseCase, logger)
	clipSubHandler := delivery.NewClipSubscriptionHandler(clipSubUseCase, logger)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService, redisClient, logger)
	rbacMiddleware := middleware.NewRBACMiddleware(rbacManager, logger)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(
		redisClient,
		logger,
		cfg.RateLimitEnabled,
		cfg.RateLimitRequests,
		cfg.RateLimitWindow,
	)

	// Initialize Idempotency Manager
	idempotencyManager := wallet.NewIdempotencyManager(redisClient, logger)

	// New Middlewares
	banCheckerMiddleware := middleware.NewBanChecker(bannedRepo, logger)
	clipQuotaMiddleware := middleware.NewClipQuotaMiddleware(clipSubUseCase, logger)

	// Setup router
	router := delivery.SetupRouter(
		handler,
		webrtcHandler,
		vodHandler,
		monetizationHandler,
		extraMonetizationHandler,
		healthHandler,
		cryptoHandler,
		pkHandler,
		moderationHandler,
		creatorTokenHandler,
		predictionHandler,
		drmHandler,
		recommendationHandler,
		clipHandler,
		kycHandler,
		clipSubHandler,
		authMiddleware,
		rbacMiddleware,
		rateLimitMiddleware,
		idempotencyManager,
		banCheckerMiddleware,
		clipQuotaMiddleware,
		logger,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort),
		Handler:      middleware.CORSWrapper(router),
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
	wsHub.Stop()

	// 2. Flush critical Redis caches or close cleanly
	if redisClient != nil {
		logger.Info("Closing Redis client...")
		_ = redisClient.Close()
	}

	// 3. Close database pools gracefully
	if db != nil {
		logger.Info("Closing PostgreSQL connection pool...")
		db.Close()
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

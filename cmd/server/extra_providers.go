package server

import (
	"github.com/google/wire"
	"nvide-live/internal/delivery"
	"nvide-live/internal/middleware"
	"nvide-live/internal/websocket"
	"nvide-live/pkg/wallet"
	"nvide-live/internal/webrtc"
	"nvide-live/internal/usecase"
	"nvide-live/internal/domain"
	workerV1 "nvide-live/internal/worker"
	"nvide-live/pkg/blockchain"
	"nvide-live/pkg/ffmpeg"
	"nvide-live/pkg/storage"
	"nvide-live/pkg/worker"
	"nvide-live/pkg/database"
	"nvide-live/pkg/redis"
	"nvide-live/pkg/config"
	"nvide-live/pkg/auth"
	"nvide-live/pkg/rbac"
	"nvide-live/pkg/broker"
	"nvide-live/pkg/duitku"
	"go.uber.org/zap"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgxPoolProvider extracts the underlying pgxpool from database.DB
func PgxPoolProvider(db *database.DB) *pgxpool.Pool {
	return db.Pool()
}

func AuthServiceProvider(cfg *config.Config) *auth.Service {
	return auth.New(
		cfg.JWTSecret,
		cfg.JWTExpiry,
		cfg.RefreshTokenExpiry,
	)
}

func RbacManagerProvider(pool *pgxpool.Pool, logger *zap.Logger) *rbac.Manager {
	return rbac.New()
}

func RateLimitMiddlewareProvider(redisClient *redis.Client, logger *zap.Logger, cfg *config.Config) *middleware.RateLimitMiddleware {
	return middleware.NewRateLimitMiddleware(
		redisClient,
		logger,
		cfg.RateLimitEnabled,
		cfg.RateLimitRequests,
		cfg.RateLimitWindow,
	)
}

var MiddlewareSet = wire.NewSet(
	middleware.NewAuthMiddleware,
	middleware.NewRBACMiddleware,
	RateLimitMiddlewareProvider,
	middleware.NewBanChecker,
	middleware.NewClipQuotaMiddleware,
	wallet.NewIdempotencyManager,
)

var WebsocketSet = wire.NewSet(
	websocket.NewHub,
	websocket.NewWaitRoomHub,
)

func BrokerProvider(redisClient *redis.Client, logger *zap.Logger) broker.Broker {
	return broker.NewHybridBroker(redisClient, logger)
}

func CryptoEncryptionKeyProvider(cfg *config.Config) []byte {
    return []byte(cfg.CryptoEncryptionKey)
}

func MicroDepositVerifyEnabledProvider(cfg *config.Config) bool {
    return cfg.MicroDepositVerifyEnabled
}

func VODUseCaseProvider(uc *usecase.VODUseCase) domain.VODUseCaseInterface {
    return uc
}

func DuitkuClientProvider(cfg *config.Config, logger *zap.Logger) *duitku.Client {
	return duitku.NewClient(&duitku.Config{
		MerchantCode: cfg.DuitkuMerchantCode,
		APIKey:       cfg.DuitkuAPIKey,
		BaseURL:      cfg.DuitkuBaseURL,
		CallbackURL:  cfg.DuitkuCallbackURL,
		ReturnURL:    cfg.DuitkuReturnURL,
	}, logger)
}

func PublisherProvider(b broker.Broker) interface{Publish(ctx context.Context, topic string, msg string) error} {
    return b
}

func CryptoMasterMnemonicProvider(cfg *config.Config) string {
    return cfg.CryptoMasterMnemonic
}

func WaitRoomHubInterfaceProvider(hub *websocket.WaitRoomHub) interface{} {
    return hub
}

func CryptoMonitorProvider(cryptoRepo domain.CryptoRepository, cryptoUseCase *usecase.CryptoUseCase, cfg *config.Config, logger *zap.Logger) *workerV1.CryptoMonitor {
    solanaClient := blockchain.NewSolanaClient(cfg.SolanaRPCURL)
	evmClient, _ := blockchain.NewEVMClient(cfg.USDTRPCURL)
    return workerV1.NewCryptoMonitor(cryptoRepo, cryptoUseCase, solanaClient, evmClient, logger)
}

func FFmpegProvider(logger *zap.Logger) *ffmpeg.FFmpeg {
	return ffmpeg.New(logger)
}

func StorageProvider(logger *zap.Logger) storage.Storage {
	return storage.NewLocalStorage("./uploads", "/uploads", logger)
}

func WorkerPoolProvider(logger *zap.Logger) *worker.WorkerPool {
	return worker.NewPool(5, logger)
}

var InfrastructureSet2 = wire.NewSet(
	PgxPoolProvider,
	AuthServiceProvider,
	RbacManagerProvider,
	BrokerProvider,
	CryptoEncryptionKeyProvider,
	MicroDepositVerifyEnabledProvider,
	VODUseCaseProvider,
	DuitkuClientProvider,
	PublisherProvider,
	CryptoMasterMnemonicProvider,
	WaitRoomHubInterfaceProvider,
	CryptoMonitorProvider,
	FFmpegProvider,
	StorageProvider,
	WorkerPoolProvider,
)

var WebRTCSet = wire.NewSet(
	webrtc.NewRoomManager,
)

var RouterSet = wire.NewSet(
	delivery.SetupRouter,
)

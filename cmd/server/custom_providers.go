package server

import (
	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
	"nvide-live/pkg/auth"
	"nvide-live/pkg/config"
	"nvide-live/pkg/rbac"
	"nvide-live/pkg/redis"

	"go.uber.org/zap"
)

func AuthUseCaseProvider(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	roleRepo domain.RoleRepository,
	authService *auth.Service,
	redisClient *redis.Client,
	rbacManager *rbac.Manager,
	logger *zap.Logger,
	cfg *config.Config,
) *usecase.AuthUseCase {
	return usecase.NewAuthUseCase(
		userRepo, tokenRepo, roleRepo, authService, redisClient, rbacManager, logger,
		cfg.JWTExpiry, cfg.RefreshTokenExpiry,
	)
}

func PushNotificationUseCaseProvider(
	pushSubscriptionRepo domain.PushSubscriptionRepository,
	cfg *config.Config,
	logger *zap.Logger,
) domain.PushNotificationUsecase {
	return usecase.NewPushNotificationUsecase(
		pushSubscriptionRepo,
		cfg.VAPIDPublicKey,
		cfg.VAPIDPrivateKey,
		cfg.VAPIDSubject,
		logger,
	)
}

package server

import (
	"github.com/google/wire"
	"go.uber.org/zap"

	"nvide-live/pkg/config"
	"nvide-live/pkg/database"
	"nvide-live/pkg/redis"
)

// ConfigProvider provides the application configuration
func ConfigProvider() *config.Config {
	return config.Load()
}

// DatabaseProvider initializes the PostgreSQL connection
func DatabaseProvider(cfg *config.Config, logger *zap.Logger) (*database.DB, error) {
	return database.New(&database.Config{
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
}

// RedisProvider initializes the Redis client
func RedisProvider(cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	return redis.New(&redis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: cfg.RedisPoolSize,
	}, logger)
}

// InfrastructureSet groups all infrastructural dependencies
var InfrastructureSet = wire.NewSet(
	ConfigProvider,
	DatabaseProvider,
	RedisProvider,
)

// AppSet is the main application set
var AppSet = wire.NewSet(
	InfrastructureSet,
	RepositorySet,
	UseCaseSet,
	HandlerSet,
	InfrastructureSet2,
	MiddlewareSet,
	WebsocketSet,
	WebRTCSet,
	RouterSet,
)

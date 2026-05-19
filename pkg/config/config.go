package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server
	ServerPort      string        `env:"SERVER_PORT" default:"8080"`
	ServerHost      string        `env:"SERVER_HOST" default:"0.0.0.0"`
	GracefulTimeout time.Duration `env:"GRACEFUL_TIMEOUT" default:"30s"`

	// Database
	DATABASE_URL string `env:"DATABASE_URL"`
	DBHost       string `env:"DB_HOST" default:"localhost"`
	DBPort       string `env:"DB_PORT" default:"5432"`
	DBUser       string `env:"DB_USER" default:"postgres"`
	DBPassword   string `env:"DB_PASSWORD" default:"postgres"`
	DBName       string `env:"DB_NAME" default:"nvide_live"`
	DBSSLMode    string `env:"DB_SSLMODE" default:"disable"`
	DBMaxConn    int    `env:"DB_MAX_CONN" default:"20"`
	DBMinConn    int    `env:"DB_MIN_CONN" default:"5"`

	// Redis
	RedisAddr     string `env:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword string `env:"REDIS_PASSWORD" default:""`
	RedisDB       int    `env:"REDIS_DB" default:"0"`
	RedisPoolSize int    `env:"REDIS_POOL_SIZE" default:"10"`

	// JWT
	JWTSecret          string        `env:"JWT_SECRET" default:"change-me-in-production"`
	JWTExpiry          time.Duration `env:"JWT_EXPIRY" default:"15m"`
	RefreshTokenExpiry time.Duration `env:"REFRESH_TOKEN_EXPIRY" default:"168h"` // 7 days

	// Bcrypt
	BcryptCost int `env:"BCRYPT_COST" default:"12"`

	// Rate Limiting
	RateLimitEnabled  bool          `env:"RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRequests int           `env:"RATE_LIMIT_REQUESTS" default:"100"`
	RateLimitWindow   time.Duration `env:"RATE_LIMIT_WINDOW" default:"1m"`

	// Logging
	LogLevel  string `env:"LOG_LEVEL" default:"info"`
	LogFormat string `env:"LOG_FORMAT" default:"json"` // json or text

	// Duitku Payment Gateway
	DuitkuMerchantCode string `env:"DUITKU_MERCHANT_CODE"`
	DuitkuAPIKey       string `env:"DUITKU_API_KEY"`
	DuitkuBaseURL      string `env:"DUITKU_BASE_URL" default:"https://sandbox.duitku.com"`
	DuitkuCallbackURL  string `env:"DUITKU_CALLBACK_URL"`
	DuitkuReturnURL    string `env:"DUITKU_RETURN_URL"`

	// Crypto and Blockchain Configurations
	CryptoEncryptionKey string `env:"CRYPTO_ENCRYPTION_KEY" default:"32-byte-long-aes-key-for-crypto"`
	SolanaRPCURL        string `env:"SOLANA_RPC_URL" default:"https://api.devnet.solana.com"`
	USDTRPCURL          string `env:"USDT_RPC_URL" default:"https://data-seed-prebsc-1-s1.binance.org:8545"`
	BTCRPCURL           string `env:"BTC_RPC_URL" default:"https://api.blockcypher.com/v1/btc/test3"`

	// KYC Region Restriction
	AllowedRegions string `env:"ALLOWED_REGIONS" default:"indonesia,philippines,filipina,thailand,malaysia,myanmar,cambodia,kamboja,vietnam,brazil,china,tiongkok,japan,jepang,india,kazakhstan"`
}

var globalConfig *Config

// Get returns the global configuration instance
func Get() *Config {
	if globalConfig == nil {
		globalConfig = Load()
	}
	return globalConfig
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{}

	// Helper to get env with default
	getEnv := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}; _ = getEnv

	// Helper to get env as int
	getEnvInt := func(key string, defaultValue int) int {
		if value := os.Getenv(key); value != "" {
			if intVal, err := strconv.Atoi(value); err == nil {
				return intVal
			}
		}
		return defaultValue
	}; _ = getEnvInt

	// Helper to get env as bool
	getEnvBool := func(key string, defaultValue bool) bool {
		if value := os.Getenv(key); value != "" {
			if boolVal, err := strconv.ParseBool(value); err == nil {
				return boolVal
			}
		}
		return defaultValue
	}; _ = getEnvBool

	// Helper to get env as duration
	getEnvDuration := func(key string, defaultValue time.Duration) time.Duration {
		if value := os.Getenv(key); value != "" {
			if durationVal, err := time.ParseDuration(value); err == nil {
				return durationVal
			}
		}
		return defaultValue
	}; _ = getEnvDuration

	// Server
	cfg.ServerPort = getEnv("SERVER_PORT", "8080")
	cfg.ServerHost = getEnv("SERVER_HOST", "0.0.0.0")
	cfg.GracefulTimeout = getEnvDuration("GRACEFUL_TIMEOUT", 30*time.Second)

	// Database
	cfg.DATABASE_URL = getEnv("DATABASE_URL", "")
	cfg.DBHost = getEnv("DB_HOST", "localhost")
	cfg.DBPort = getEnv("DB_PORT", "5432")
	cfg.DBUser = getEnv("DB_USER", "postgres")
	cfg.DBPassword = getEnv("DB_PASSWORD", "postgres")
	cfg.DBName = getEnv("DB_NAME", "nvide_live")
	cfg.DBSSLMode = getEnv("DB_SSLMODE", "disable")
	cfg.DBMaxConn = getEnvInt("DB_MAX_CONN", 20)
	cfg.DBMinConn = getEnvInt("DB_MIN_CONN", 5)

	// Redis
	cfg.RedisAddr = getEnv("REDIS_ADDR", "localhost:6379")
	cfg.RedisPassword = getEnv("REDIS_PASSWORD", "")
	cfg.RedisDB = getEnvInt("REDIS_DB", 0)
	cfg.RedisPoolSize = getEnvInt("REDIS_POOL_SIZE", 10)

	// JWT
	cfg.JWTSecret = getEnv("JWT_SECRET", "change-me-in-production")
	cfg.JWTExpiry = getEnvDuration("JWT_EXPIRY", 15*time.Minute)
	cfg.RefreshTokenExpiry = getEnvDuration("REFRESH_TOKEN_EXPIRY", 168*time.Hour)

	// Bcrypt
	cfg.BcryptCost = getEnvInt("BCRYPT_COST", 12)

	// Rate Limiting
	cfg.RateLimitEnabled = getEnvBool("RATE_LIMIT_ENABLED", true)
	cfg.RateLimitRequests = getEnvInt("RATE_LIMIT_REQUESTS", 100)
	cfg.RateLimitWindow = getEnvDuration("RATE_LIMIT_WINDOW", 1*time.Minute)

	// Logging
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")
	cfg.LogFormat = getEnv("LOG_FORMAT", "json")

	// Duitku
	cfg.DuitkuMerchantCode = getEnv("DUITKU_MERCHANT_CODE", "")
	cfg.DuitkuAPIKey = getEnv("DUITKU_API_KEY", "")
	cfg.DuitkuBaseURL = getEnv("DUITKU_BASE_URL", "https://sandbox.duitku.com")
	cfg.DuitkuCallbackURL = getEnv("DUITKU_CALLBACK_URL", "")
	cfg.DuitkuReturnURL = getEnv("DUITKU_RETURN_URL", "")

	// Crypto & Blockchain
	cfg.CryptoEncryptionKey = getEnv("CRYPTO_ENCRYPTION_KEY", "32-byte-long-aes-key-for-crypto")
	cfg.SolanaRPCURL = getEnv("SOLANA_RPC_URL", "https://api.devnet.solana.com")
	cfg.USDTRPCURL = getEnv("USDT_RPC_URL", "https://data-seed-prebsc-1-s1.binance.org:8545")
	cfg.BTCRPCURL = getEnv("BTC_RPC_URL", "https://api.blockcypher.com/v1/btc/test3")

	// KYC Region Restriction
	cfg.AllowedRegions = getEnv("ALLOWED_REGIONS", "indonesia,philippines,filipina,thailand,malaysia,myanmar,cambodia,kamboja,vietnam,brazil,china,tiongkok,japan,jepang,india,kazakhstan")

	globalConfig = cfg
	return cfg
}

// GetDBConnectionString returns PostgreSQL connection string
func (c *Config) GetDBConnectionString() string {
	return "postgres://" + c.DBUser + ":" + c.DBPassword +
		"@" + c.DBHost + ":" + c.DBPort +
		"/" + c.DBName + "?sslmode=" + c.DBSSLMode
}

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Manager handles primary and replica database connections
type Manager struct {
	primary *pgxpool.Pool
	replica *pgxpool.Pool
	logger  *zap.Logger
}

// NewManager creates a new database manager with primary and optional replica
func NewManager(primaryCfg, replicaCfg *Config, logger *zap.Logger) (*Manager, error) {
	primary, err := createPool(primaryCfg, logger, "primary")
	if err != nil {
		return nil, err
	}

	var replica *pgxpool.Pool
	if replicaCfg != nil && replicaCfg.Host != "" {
		replica, err = createPool(replicaCfg, logger, "replica")
		if err != nil {
			logger.Warn("Failed to connect to replica, using primary for all queries", zap.Error(err))
			replica = primary
		}
	} else {
		replica = primary
	}

	return &Manager{
		primary: primary,
		replica: replica,
		logger:  logger,
	}, nil
}

func createPool(cfg *Config, logger *zap.Logger, name string) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s database config: %w", name, err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConn)
	poolConfig.MinConns = int32(cfg.MinConn)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s connection pool: %w", name, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping %s database: %w", name, err)
	}

	logger.Info("Database connection established",
		zap.String("type", name),
		zap.String("host", cfg.Host),
	)

	return pool, nil
}

// Primary returns the primary pool for writes
func (m *Manager) Primary() *pgxpool.Pool {
	return m.primary
}

// Replica returns the replica pool for reads
func (m *Manager) Replica() *pgxpool.Pool {
	return m.replica
}

// Close closes both pools
func (m *Manager) Close() {
	m.primary.Close()
	if m.replica != m.primary {
		m.replica.Close()
	}
}

// Health checks health of both pools
func (m *Manager) Health(ctx context.Context) error {
	if err := m.primary.Ping(ctx); err != nil {
		return fmt.Errorf("primary: %w", err)
	}
	if m.replica != m.primary {
		if err := m.replica.Ping(ctx); err != nil {
			return fmt.Errorf("replica: %w", err)
		}
	}
	return nil
}

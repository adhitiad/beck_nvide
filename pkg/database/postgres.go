package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DB represents PostgreSQL connection pool
type DB struct {
	pool *pgxpool.Pool
}

// Config holds database configuration
type Config struct {
	DATABASE_URL string
	Host         string
	Port         string
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxConn      int
	MinConn      int
}

// New creates a new database connection pool
func New(cfg *Config, logger *zap.Logger) (*DB, error) {

	connStr := cfg.DATABASE_URL
	if connStr == "" {
		connStr = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode,
		)
	}

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConn)
	poolConfig.MinConns = int32(cfg.MinConn)
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.String("database", cfg.DBName),
	)

	return &DB{pool: pool}, nil
}

// Pool returns the pgxpool.Pool
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// QueryContext executes a query that returns rows
func (db *DB) QueryContext(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRowContext executes a query that returns at most one row
func (db *DB) QueryRowContext(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// ExecContext executes a query that doesn't return rows
func (db *DB) ExecContext(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// Begin starts a transaction
func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.pool.Close()
}

// Health checks database health
func (db *DB) Health(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

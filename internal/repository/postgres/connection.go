package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds all configuration parameters for the PostgreSQL connection pool.
// Values are loaded from environment variables via the config package.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string

	// Pool tuning — these defaults are optimized for a production API workload.
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultConfig returns a Config with production-safe defaults.
// Override individual fields after calling this.
func DefaultConfig() Config {
	return Config{
		SSLMode:           "disable",
		MaxConns:          25,
		MinConns:          5,
		MaxConnLifetime:   30 * time.Minute,
		MaxConnIdleTime:   5 * time.Minute,
		HealthCheckPeriod: 1 * time.Minute,
	}
}

// DSN builds the PostgreSQL Data Source Name (connection string).
func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

// NewPool creates and validates a new pgxpool connection pool.
//
// This function:
//  1. Builds the connection pool with the provided configuration.
//  2. Pings the database to verify connectivity before returning.
//  3. Returns an error if the connection cannot be established within the context deadline.
//
// The caller is responsible for calling pool.Close() when done (typically on server shutdown).
//
// Example:
//
//	pool, err := postgres.NewPool(ctx, cfg)
//	if err != nil {
//	    log.Fatal("failed to connect to database:", err)
//	}
//	defer pool.Close()
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsing database DSN: %w", err)
	}

	// Apply pool tuning parameters.
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Verify the connection is live before returning.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database at %s:%d: %w", cfg.Host, cfg.Port, err)
	}

	return pool, nil
}

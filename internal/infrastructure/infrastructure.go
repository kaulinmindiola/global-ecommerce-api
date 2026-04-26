package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaulinmindiola/global-ecommerce-api/config"
	"github.com/redis/go-redis/v9"
)

// ConnectPostgres creates a validated pgxpool connection pool.
//
// Retry strategy: attempts up to maxRetries times with exponential backoff.
// This handles the Docker startup race condition where the API container
// starts before PostgreSQL is fully ready to accept connections.
func ConnectPostgres(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parsing database DSN: %w", err)
	}

	// Apply pool tuning from configuration.
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	const maxRetries = 5
	var pool *pgxpool.Pool

	for attempt := 1; attempt <= maxRetries; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolCfg)
		if err == nil {
			// Verify the connection is actually live, not just initialised.
			if pingErr := pool.Ping(ctx); pingErr == nil {
				slog.Info("PostgreSQL connected",
					"host", cfg.Host,
					"port", cfg.Port,
					"database", cfg.DBName,
					"max_conns", cfg.MaxConns,
				)
				return pool, nil
			}
			pool.Close()
		}

		if attempt == maxRetries {
			break
		}

		// Exponential backoff: 1s, 2s, 4s, 8s between retries.
		backoff := time.Duration(1<<uint(attempt-1)) * time.Second
		slog.Warn("PostgreSQL connection attempt failed, retrying",
			"attempt", attempt,
			"max_retries", maxRetries,
			"backoff", backoff,
			"error", err,
		)
		time.Sleep(backoff)
	}

	return nil, fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %w", maxRetries, err)
}

// ConnectRedis creates and validates a Redis client connection.
//
// Uses the same retry strategy as ConnectPostgres to handle Docker startup
// race conditions gracefully.
func ConnectRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,

		// Connection pool settings aligned with the PostgreSQL pool for consistency.
		PoolSize:        10,
		MinIdleConns:    2,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,

		// Timeouts prevent hanging connections from blocking request handlers.
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	const maxRetries = 5

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := client.Ping(ctx).Err(); err == nil {
			slog.Info("Redis connected",
				"addr", cfg.Addr(),
				"db", cfg.DB,
			)
			return client, nil
		} else if attempt == maxRetries {
			client.Close()
			return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, err)
		}

		backoff := time.Duration(1<<uint(attempt-1)) * time.Second
		slog.Warn("Redis connection attempt failed, retrying",
			"attempt", attempt,
			"backoff", backoff,
		)
		time.Sleep(backoff)
	}

	return client, nil
}

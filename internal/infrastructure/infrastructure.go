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
	poolCfg.MaxConns = cfg.Pool.MaxConns
	poolCfg.MinConns = cfg.Pool.MinConns
	poolCfg.MaxConnLifetime = cfg.Pool.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.Pool.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.Pool.HealthCheckPeriod

	const maxRetries = 5
	var pool *pgxpool.Pool

	for attempt := 1; attempt <= maxRetries; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolCfg)
		if err == nil {
			// Verify real DB connectivity
			pingErr := pool.Ping(ctx)
			if pingErr == nil {
				slog.Info("PostgreSQL connected",
					"host", cfg.Host,
					"port", cfg.Port,
					"database", cfg.DBName,
					"max_conns", cfg.Pool.MaxConns,
				)
				return pool, nil
			}

			// IMPORTANT: preserve real ping error
			err = pingErr
			pool.Close()
		}

		if attempt == maxRetries {
			break
		}

		backoff := time.Duration(1<<uint(attempt-1)) * time.Second

		slog.Warn("PostgreSQL connection attempt failed, retrying",
			"attempt", attempt,
			"max_retries", maxRetries,
			"backoff", backoff,
			"error", err,
		)

		time.Sleep(backoff)
	}

	return nil, fmt.Errorf(
		"failed to connect to PostgreSQL after %d attempts: %w",
		maxRetries,
		err,
	)
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

		// Connection pool settings dynamically loaded from configuration.
		PoolSize:        cfg.Pool.PoolSize,
		MinIdleConns:    cfg.Pool.MinIdleConns,
		ConnMaxLifetime: cfg.Pool.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Pool.ConnMaxIdleTime,

		// Timeouts dynamically loaded from configuration.
		DialTimeout:  cfg.Pool.DialTimeout,
		ReadTimeout:  cfg.Pool.ReadTimeout,
		WriteTimeout: cfg.Pool.WriteTimeout,
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

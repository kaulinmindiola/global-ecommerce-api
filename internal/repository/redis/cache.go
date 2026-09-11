package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"github.com/redis/go-redis/v9"
)

// ErrCacheMiss is returned when a requested key is not found in the cache.
var ErrCacheMiss = errors.New("cache: key not found")

// cacheRepository implements repository.CacheRepository using Redis.
type cacheRepository struct {
	client *redis.Client
}

// NewCacheRepository creates a new Redis-backed CacheRepository.
// The client should be created externally and injected here (dependency inversion).
func NewCacheRepository(client *redis.Client) repository.CacheRepository {
	return &cacheRepository{client: client}
}

// Set serializes the value to JSON and stores it in Redis with a TTL.
// If ttl is 0, the key will never expire (use with care).
//
// Design note: JSON serialization is used for portability. For high-throughput
// scenarios, consider MessagePack or Protocol Buffers as a future optimization.
func (r *cacheRepository) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("serializing value for cache key %q: %w", key, err)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("setting cache key %q: %w", key, err)
	}

	return nil
}

// Get retrieves a value from Redis and deserializes it into dest.
// dest must be a pointer to the expected type (e.g., &domain.Product{}).
// Returns ErrCacheMiss if the key does not exist or has expired.
func (r *cacheRepository) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("getting cache key %q: %w", key, err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("deserializing value for cache key %q: %w", key, err)
	}

	return nil
}

// Delete removes a single key from Redis.
// If the key does not exist, no error is returned (idempotent).
func (r *cacheRepository) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("deleting cache key %q: %w", key, err)
	}
	return nil
}

// DeleteByPattern removes all keys matching a glob-style pattern using SCAN + DEL.
//
// IMPORTANT: This uses SCAN (non-blocking, cursor-based) instead of KEYS,
// which would block the Redis server on large datasets — a critical distinction
// for production systems. SCAN iterates in small batches (100 keys per call).
//
// Example patterns:
//
//	"product:*"     — invalidate all product cache entries
//	"user:*:orders" — invalidate cached order lists for all users
func (r *cacheRepository) DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	const scanBatchSize = 100

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, scanBatchSize).Result()
		if err != nil {
			return fmt.Errorf("scanning keys with pattern %q: %w", pattern, err)
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("deleting keys matching %q: %w", pattern, err)
			}
		}

		cursor = nextCursor
		// SCAN returns cursor=0 when the full iteration is complete.
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Exists checks whether a key is present in the cache.
// Returns false without error if the key has expired or never existed.
func (r *cacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("checking existence of cache key %q: %w", key, err)
	}
	return count > 0, nil
}

// Increment atomically increments the integer value stored at key by 1.
// If the key does not exist, it is initialized to 0 before incrementing.
// Useful for rate limiting counters and request tracking.
func (r *cacheRepository) Increment(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incrementing cache key %q: %w", key, err)
	}
	return val, nil
}

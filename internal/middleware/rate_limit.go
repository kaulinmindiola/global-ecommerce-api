package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig holds the configuration for the rate limiter.
type RateLimitConfig struct {
	// RequestsPerWindow is the maximum number of requests allowed per client per window.
	RequestsPerWindow int

	// Window is the duration of the sliding time window.
	Window time.Duration

	// KeyPrefix is the Redis key prefix for rate limit counters.
	KeyPrefix string
}

// DefaultRateLimitConfig returns the standard rate limit:
// 100 requests per minute per IP address.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerWindow: 100,
		Window:            time.Minute,
		KeyPrefix:         "rate_limit",
	}
}

// RateLimit returns a rate limiting middleware using the default configuration.
func RateLimit(redisClient *redis.Client) func(http.Handler) http.Handler {
	return RateLimitWithConfig(redisClient, DefaultRateLimitConfig())
}

// RateLimitWithConfig returns a rate limiting middleware with custom configuration.
//
// Algorithm: Fixed window counter using Redis INCR + EXPIRE.
//
// For each request:
//  1. Extract the client IP address.
//  2. Build a Redis key: "{prefix}:{ip}:{window_start_unix}".
//  3. INCR the counter atomically.
//  4. On first request in the window, set the key TTL to the window duration.
//  5. If the counter exceeds the limit, return 429 Too Many Requests.
//
// Redis key example: "rate_limit:192.168.1.1:1708000000"
//
// The fixed window approach is simpler than a sliding window and sufficient
// for an MVP. The counter resets cleanly at the start of each window.
// For stricter rate limiting, upgrade to a sliding window with a sorted set.
//
// Rate limit headers (standard X-RateLimit-* headers):
//   - X-RateLimit-Limit     : maximum requests per window
//   - X-RateLimit-Remaining : requests remaining in current window
//   - X-RateLimit-Reset     : Unix timestamp when the window resets
//   - Retry-After           : seconds to wait (only on 429)
//
// RateLimitWithConfig returns a rate limiting middleware with custom configuration.
// ... (comentarios originales) ...
func RateLimitWithConfig(redisClient *redis.Client, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 🛡️ PROTECCIÓN SENIOR (Fail-Open):
			// Si el cliente de Redis es nil (ej. entorno de tests o fallo de inyección),
			// saltamos el Rate Limit y permitimos que la petición continúe.
			if redisClient == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := extractClientIP(r)

			// Build the window-scoped Redis key.
			windowStart := time.Now().Truncate(cfg.Window).Unix()
			key := fmt.Sprintf("%s:%s:%d", cfg.KeyPrefix, ip, windowStart)

			ctx := r.Context()

			// Atomically increment and fetch the counter value.
			count, err := incrementCounter(ctx, redisClient, key, cfg.Window)
			if err != nil {
				// If Redis is unavailable, fail open (allow the request).
				slog.Warn("rate limiter: Redis unavailable, allowing request",
					"error", err,
					"ip", ip,
				)
				next.ServeHTTP(w, r)
				return
			}

			// ... [EL RESTO DE TU FUNCIÓN SE MANTIENE EXACTAMENTE IGUAL] ...

			remaining := cfg.RequestsPerWindow - int(count)
			if remaining < 0 {
				remaining = 0
			}

			resetAt := time.Now().Truncate(cfg.Window).Add(cfg.Window).Unix()

			// Always set rate limit headers so clients can implement backoff.
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt, 10))

			if int(count) > cfg.RequestsPerWindow {
				retryAfter := time.Until(time.Unix(resetAt, 0)).Seconds()
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter)+1))

				requestID := RequestIDFromContext(r.Context())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				fmt.Fprintf(w,
					`{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Too many requests. Try again in %d seconds.","timestamp":%q,"path":%q,"request_id":%q}}`,
					int(retryAfter)+1,
					time.Now().UTC().Format(time.RFC3339),
					r.URL.Path,
					requestID,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// incrementCounter atomically increments a Redis counter and sets its TTL
// on first creation. Uses a pipeline to execute both commands in one round-trip.
func incrementCounter(ctx context.Context, client *redis.Client, key string, ttl time.Duration) (int64, error) {
	pipe := client.Pipeline()
	incr := pipe.Incr(ctx, key)
	// EXPIRE is ignored if the key already has a TTL — safe to call every time.
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("rate limit pipeline: %w", err)
	}

	return incr.Val(), nil
}

// extractClientIP returns the real client IP, respecting reverse proxy headers.
// Priority: X-Forwarded-For → X-Real-IP → RemoteAddr
func extractClientIP(r *http.Request) string {
	// X-Forwarded-For may contain a chain: "client, proxy1, proxy2"
	// The leftmost IP is the original client.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := splitAndTrim(xff, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr and strip the port.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// splitAndTrim splits a string by sep and trims whitespace from each element.
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		if trimmed := trimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

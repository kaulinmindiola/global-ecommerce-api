package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORSConfig holds CORS policy settings.
// In production, AllowedOrigins must be an explicit allowlist —
// never a wildcard when the API serves authenticated requests.
type CORSConfig struct {
	// AllowedOrigins is the list of origins permitted to make cross-site requests.
	// Use "*" for development only. In production, list exact origins:
	// ["https://app.mystore.com", "https://admin.mystore.com"]
	AllowedOrigins []string

	// AllowedMethods lists the HTTP methods allowed in CORS requests.
	AllowedMethods []string

	// AllowedHeaders lists the request headers the browser is allowed to send.
	AllowedHeaders []string

	// ExposedHeaders lists the response headers the browser is allowed to read.
	ExposedHeaders []string

	// AllowCredentials controls the Access-Control-Allow-Credentials header.
	// Set to true when the frontend sends cookies or Authorization headers.
	AllowCredentials bool

	// MaxAge is the number of seconds the browser can cache a preflight response.
	// Reduces the number of OPTIONS requests in production.
	MaxAge string
}

// DefaultCORSConfig returns a production-safe CORS configuration.
// Origins are read from the CORS_ALLOWED_ORIGINS environment variable
// (comma-separated list). Falls back to "*" in development.
func DefaultCORSConfig() CORSConfig {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string

	if origins != "" {
		for _, o := range strings.Split(origins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	} else {
		// Development fallback — never use wildcard in production with credentials.
		allowedOrigins = []string{"*"}
	}

	return CORSConfig{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Authorization",
			"Content-Type",
			"Accept",
			"X-Request-ID",
			"X-Requested-With",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"Content-Length",
		},
		AllowCredentials: true,
		MaxAge:           "86400", // 24 hours
	}
}

// CORS returns a CORS middleware using the default environment-driven configuration.
// Use CORSWithConfig for custom settings.
func CORS(next http.Handler) http.Handler {
	return CORSWithConfig(DefaultCORSConfig())(next)
}

// CORSWithConfig returns a CORS middleware with explicit configuration.
// This is the preferred form when you need fine-grained control.
func CORSWithConfig(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set CORS headers on every response (including non-preflight).
			if isOriginAllowed(origin, cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if containsWildcard(cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
			w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
			w.Header().Set("Access-Control-Max-Age", cfg.MaxAge)

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// Preflight requests (OPTIONS) are answered immediately.
			// They must NOT reach the actual handler.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed returns true if the given origin is in the allowlist.
func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
	}
	return false
}

// containsWildcard returns true if the allowlist contains "*".
func containsWildcard(allowed []string) bool {
	for _, a := range allowed {
		if a == "*" {
			return true
		}
	}
	return false
}

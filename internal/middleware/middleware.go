package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// contextKey is an unexported type for context keys defined in this package.
type contextKey string

const (
	contextKeyUserID    contextKey = "user_id"
	contextKeyRequestID contextKey = "request_id"
)

// ─────────────────────────────────────────────
// RequestID middleware
// ─────────────────────────────────────────────

// RequestID generates a unique trace ID per request and attaches it to:
//   - The request context (for error responses and log correlation)
//   - The X-Request-ID response header (for client-side tracing)
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = "req_" + uuid.New().String()[:8]
		}

		ctx := context.WithValue(r.Context(), contextKeyRequestID, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ─────────────────────────────────────────────
// Logger middleware
// ─────────────────────────────────────────────

// responseWriter wraps http.ResponseWriter to capture the status code
// written by the handler, since http.ResponseWriter does not expose it.
type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

// Logger emits a structured log line for every HTTP request.
// Format follows the pattern used by production systems: method, path,
// status, latency, response size, and request ID.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := newResponseWriter(w)

		next.ServeHTTP(wrapped, r)

		requestID, _ := r.Context().Value(contextKeyRequestID).(string)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"latency_ms", time.Since(start).Milliseconds(),
			"bytes", wrapped.bytes,
			"request_id", requestID,
			"remote_ip", r.RemoteAddr,
		)
	})
}

// ─────────────────────────────────────────────
// Recoverer middleware
// ─────────────────────────────────────────────

// Recoverer catches any panic in downstream handlers and returns a 500
// instead of crashing the server. The stack trace is logged but never
// sent to the client (to avoid leaking internal details).
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", rec),
					"stack", string(debug.Stack()),
				)
				http.Error(w,
					`{"error":{"code":"SYSTEM_INTERNAL_ERROR","message":"An unexpected error occurred"}}`,
					http.StatusInternalServerError,
				)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────
// CORS middleware
// ─────────────────────────────────────────────

// CORS sets permissive CORS headers suitable for a development/MVP API.
// In production, replace the wildcard origin with an allowlist of trusted domains.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight requests immediately without hitting the handler.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ─────────────────────────────────────────────
// Authenticate middleware
// ─────────────────────────────────────────────

// Authenticate validates the JWT Bearer token in the Authorization header
// and stores the authenticated user's UUID in the request context.
//
// Errors returned follow the spec error code catalog (page 4):
//   - No header      → AUTH_TOKEN_MISSING  (401)
//   - Invalid/expired → AUTH_TOKEN_INVALID  (401)
func Authenticate(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeAuthError(w, r, http.StatusUnauthorized,
					"AUTH_TOKEN_MISSING", "Authorization header is required")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeAuthError(w, r, http.StatusUnauthorized,
					"AUTH_TOKEN_INVALID", "Authorization header format must be 'Bearer {token}'")
				return
			}

			rawToken := strings.TrimSpace(parts[1])
			claims, err := authService.ValidateAccessToken(r.Context(), rawToken)
			if err != nil {
				writeAuthError(w, r, http.StatusUnauthorized,
					"AUTH_TOKEN_INVALID", "Token is invalid or has expired")
				return
			}

			// Inject the authenticated user's UUID into the context.
			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ─────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────

// writeAuthError writes a minimal auth error response.
// We cannot import the handler package here (circular dep), so we write JSON directly.
func writeAuthError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	requestID, _ := r.Context().Value(contextKeyRequestID).(string)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	fmt.Fprintf(w, `{"error":{"code":%q,"message":%q,"timestamp":%q,"path":%q,"request_id":%q}}`,
		code, message,
		time.Now().UTC().Format(time.RFC3339),
		r.URL.Path,
		requestID,
	)
}

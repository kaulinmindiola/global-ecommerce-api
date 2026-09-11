package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is the unexported type for all context keys in this package.
// Using a dedicated type prevents collisions with keys from other packages.
type contextKey string

const (
	contextKeyRequestID contextKey = "request_id"
	contextKeyUserID    contextKey = "user_id"
)

// RequestID generates a unique trace ID for every incoming request.
//
// Behaviour:
//   - If the client sends an X-Request-ID header, that value is reused.
//     This supports distributed tracing where a gateway sets the ID upstream.
//   - Otherwise a new ID is generated: "req_" + first 8 chars of a UUID v4.
//
// The ID is stored in two places so all downstream code can access it:
//  1. The request context  — for log correlation inside handlers.
//  2. The X-Request-ID response header — so clients can correlate their logs.
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

// RequestIDFromContext retrieves the request trace ID from a context.
// Returns an empty string if no ID was set (e.g. in tests that skip middleware).
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKeyRequestID).(string)
	return id
}

// UserIDFromContext retrieves the authenticated user's UUID string from context.
// Returns an empty string if the user is not authenticated.
// Set by the Auth middleware after successful JWT validation.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKeyUserID).(string)
	return id
}

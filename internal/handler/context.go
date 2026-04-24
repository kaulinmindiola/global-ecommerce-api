package handler

import (
	"context"

	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys in this package.
// Using a dedicated type prevents collisions with keys from other packages.
type contextKey string

const (
	contextKeyUserID    contextKey = "user_id"
	contextKeyRequestID contextKey = "request_id"
)

// withUserID stores an authenticated user's UUID in the request context.
// Called by the JWT authentication middleware after token validation.
func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, contextKeyUserID, id)
}

// userIDFromContext retrieves the authenticated user's UUID from context.
// Returns uuid.Nil if no user ID was set — handlers must check for this.
func userIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(contextKeyUserID).(uuid.UUID)
	return id
}

// withRequestID stores a unique request trace ID in the context.
// Set by the RequestID middleware at the start of every request.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, id)
}

// requestIDFromContext retrieves the request trace ID from context.
// Included in all error responses for log correlation (spec page 3).
func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKeyRequestID).(string)
	return id
}

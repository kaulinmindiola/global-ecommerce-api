package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// Authenticate is a middleware factory that validates JWT Bearer tokens.
// It requires the AuthService to validate tokens (signature + expiry + blocklist).
//
// On success: the authenticated user's ID is stored in the request context
// and the request proceeds to the next handler.
//
// On failure: a 401 JSON error response is written and the request is stopped.
// The error codes match the catalog in the HTTP spec:
//
//	No Authorization header  → AUTH_TOKEN_MISSING  (401)
//	Malformed header format  → AUTH_TOKEN_INVALID  (401)
//	Invalid/expired token    → AUTH_TOKEN_INVALID  (401)
//
// Usage in router:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(middleware.Authenticate(authService))
//	    r.Post("/orders", orderHandler.Create)
//	})
func Authenticate(authSvc service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Step 1: Extract the Authorization header.
			header := r.Header.Get("Authorization")
			if header == "" {
				writeAuthError(w, r, "AUTH_TOKEN_MISSING", "Authorization header is required")
				return
			}

			// Step 2: Validate the "Bearer {token}" format.
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeAuthError(w, r, "AUTH_TOKEN_INVALID",
					"Authorization header format must be: Bearer {token}")
				return
			}

			rawToken := strings.TrimSpace(parts[1])
			if rawToken == "" {
				writeAuthError(w, r, "AUTH_TOKEN_MISSING", "Bearer token is empty")
				return
			}

			// Step 3: Validate the token via AuthService
			// (checks signature, expiry, and Redis blocklist for logged-out tokens).
			claims, err := authSvc.ValidateAccessToken(r.Context(), rawToken)
			if err != nil {
				writeAuthError(w, r, "AUTH_TOKEN_INVALID", "Token is invalid or has expired")
				return
			}

			// Step 4: Store the validated user ID in the request context.
			// Downstream handlers retrieve it with middleware.UserIDFromContext(ctx).
			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// writeAuthError writes a minimal 401 JSON error response.
// We write the JSON directly here to avoid importing the response or handler
// packages (which would create circular dependencies).
func writeAuthError(w http.ResponseWriter, r *http.Request, code, message string) {
	requestID := RequestIDFromContext(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	fmt.Fprintf(w,
		`{"error":{"code":%q,"message":%q,"timestamp":%q,"path":%q,"request_id":%q}}`,
		code,
		message,
		time.Now().UTC().Format(time.RFC3339),
		r.URL.Path,
		requestID,
	)
}

package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// defaultMaxBodyBytes is the default request body size limit: 1 MB.
	// Prevents large payloads from exhausting server memory.
	defaultMaxBodyBytes = 1 << 20 // 1 MB
)

// ValidateBody enforces two rules on every request that carries a body:
//
//  1. Content-Type must be "application/json" — this API is JSON-only.
//  2. Body size must not exceed maxBytes — prevents memory exhaustion attacks.
//
// These checks happen before the handler reads the body, so malformed
// requests are rejected cheaply without involving any business logic.
//
// Apply this middleware only to routes that expect a request body
// (POST, PUT, PATCH). It is intentionally NOT applied globally to avoid
// rejecting GET/DELETE requests that legitimately have no body.
//
// Usage in router:
//
//	r.With(middleware.ValidateBody).Post("/orders", orderHandler.Create)
func ValidateBody(next http.Handler) http.Handler {
	return ValidateBodyWithLimit(defaultMaxBodyBytes)(next)
}

// ValidateBodyWithLimit returns a body validation middleware with a custom size limit.
// Use this when a specific endpoint requires a different limit (e.g. file upload endpoints).
func ValidateBodyWithLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only validate requests that are expected to carry a body.
			if r.Method == http.MethodPost ||
				r.Method == http.MethodPut ||
				r.Method == http.MethodPatch {

				// ── Rule 1: Content-Type must be application/json ─────────
				contentType := r.Header.Get("Content-Type")
				if !isJSONContentType(contentType) {
					writeBodyError(w, r, http.StatusUnsupportedMediaType,
						"UNSUPPORTED_MEDIA_TYPE",
						"Content-Type must be application/json",
					)
					return
				}

				// ── Rule 2: Body size must not exceed the limit ───────────
				// http.MaxBytesReader wraps the body and returns a 413 error
				// if the client tries to send more than maxBytes.
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isJSONContentType returns true if the Content-Type header indicates JSON.
// Accepts "application/json" and "application/json; charset=utf-8".
func isJSONContentType(contentType string) bool {
	// Split on ";" to handle "application/json; charset=utf-8" gracefully.
	mediaType := strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	return strings.EqualFold(mediaType, "application/json")
}

// writeBodyError writes a lightweight JSON error for body validation failures.
// Avoids importing response/errors packages to prevent circular dependencies.
func writeBodyError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	requestID := RequestIDFromContext(r.Context())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	fmt.Fprintf(w,
		`{"error":{"code":%q,"message":%q,"timestamp":%q,"path":%q,"request_id":%q}}`,
		code,
		message,
		time.Now().UTC().Format(time.RFC3339),
		r.URL.Path,
		requestID,
	)
}

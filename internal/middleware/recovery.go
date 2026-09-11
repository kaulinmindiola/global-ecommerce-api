package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Recovery catches any panic that occurs in downstream handlers and converts
// it into a structured 500 Internal Server Error response.
//
// Without this middleware, an unhandled panic would crash the entire server
// process — not just the single failing request. In Go HTTP servers, each
// request runs in its own goroutine, so a panic only kills that goroutine,
// but the net/http default behaviour still closes the connection abruptly.
//
// What this middleware does:
//  1. Defers a recovery function before calling the next handler.
//  2. On panic: logs the error AND the full stack trace for post-mortem debugging.
//  3. Writes a safe, generic 500 response that never leaks internal details.
//
// The stack trace is logged server-side but never sent to the client — this is
// critical for security. A stack trace in a response reveals internal paths,
// package names, and sometimes secrets.
//
// Place this AFTER Logger in the middleware stack so panics are also captured
// in the request log with the correct 500 status code.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := RequestIDFromContext(r.Context())

				// Log the full panic context for server-side debugging.
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", rec),
					"method", r.Method,
					"path", r.URL.Path,
					"request_id", requestID,
					"stack", string(debug.Stack()),
				)

				// Write a safe generic error response.
				// We cannot use the response package here (circular import risk),
				// so we write the JSON envelope manually.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)

				fmt.Fprintf(w,
					`{"error":{"code":"SYSTEM_INTERNAL_ERROR","message":"An unexpected error occurred","timestamp":%q,"path":%q,"request_id":%q}}`,
					time.Now().UTC().Format(time.RFC3339),
					r.URL.Path,
					requestID,
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

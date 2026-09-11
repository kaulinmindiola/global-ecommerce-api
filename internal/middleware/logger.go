package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseRecorder wraps http.ResponseWriter to capture the HTTP status code
// and response body size written by downstream handlers.
// The standard http.ResponseWriter does not expose these values after the fact.
type responseRecorder struct {
	http.ResponseWriter
	status        int
	bytesWritten  int
	headerWritten bool
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		status:         http.StatusOK, // default if WriteHeader is never called
	}
}

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.headerWritten {
		rr.status = code
		rr.headerWritten = true
		rr.ResponseWriter.WriteHeader(code)
	}
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.headerWritten {
		rr.WriteHeader(http.StatusOK)
	}
	n, err := rr.ResponseWriter.Write(b)
	rr.bytesWritten += n
	return n, err
}

// Logger emits one structured JSON log line per HTTP request.
//
// Every log entry includes:
//   - method, path, status — for request identification
//   - latency_ms           — for performance monitoring
//   - bytes                — for bandwidth tracking
//   - request_id           — for log correlation with error responses
//   - remote_ip            — for security auditing
//   - user_agent           — for client analytics
//
// Log level is chosen based on the response status code:
//   - 5xx → ERROR  (server-side fault)
//   - 4xx → WARN   (client-side issue, still worth monitoring)
//   - 2xx/3xx → INFO (normal traffic)
//
// This must be placed AFTER RequestID in the middleware stack so the
// request_id is available in the context when Logger runs.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		latencyMs := time.Since(start).Milliseconds()
		requestID := RequestIDFromContext(r.Context())

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"latency_ms", latencyMs,
			"bytes", recorder.bytesWritten,
			"request_id", requestID,
			"remote_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		}

		// Include query string only when present to keep logs concise.
		if r.URL.RawQuery != "" {
			attrs = append(attrs, "query", r.URL.RawQuery)
		}

		switch {
		case recorder.status >= 500:
			slog.Error("request completed", attrs...)
		case recorder.status >= 400:
			slog.Warn("request completed", attrs...)
		default:
			slog.Info("request completed", attrs...)
		}
	})
}

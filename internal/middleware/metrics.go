package middleware

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/metrics"
)

// pathNormalisationRules replaces dynamic path segments with placeholders.
// This prevents label cardinality explosion in Prometheus:
// /api/v1/products/abc-123 → /api/v1/products/{id}
var pathNormalisationRules = []*regexp.Regexp{
	// UUID v4
	regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`),
	// Numeric IDs
	regexp.MustCompile(`\b\d{4,}\b`),
	// Order numbers (ORD-2025-XXXXXX)
	regexp.MustCompile(`ORD-\d{4}-[A-Z0-9]+`),
}

// normalisePath replaces dynamic segments in a URL path with placeholders
// so Prometheus label cardinality remains bounded.
func normalisePath(path string) string {
	for _, re := range pathNormalisationRules {
		path = re.ReplaceAllString(path, "{id}")
	}
	return path
}

// metricsResponseWriter wraps http.ResponseWriter to capture status and size.
type metricsResponseWriter struct {
	http.ResponseWriter
	status        int
	bytesWritten  int
	headerWritten bool
}

func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	if !rw.headerWritten {
		rw.status = code
		rw.headerWritten = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// PrometheusMetrics returns a middleware that instruments every HTTP request
// with Prometheus metrics: request count, duration, response size, and in-flight gauge.
//
// Path normalisation ensures that /products/abc-123 and /products/def-456
// are counted under the same label /products/{id}, keeping cardinality bounded.
//
// Place this middleware AFTER RequestID and Logger so tracing is consistent.
func PrometheusMetrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip the /metrics endpoint itself to avoid meta-instrumentation noise.
			if strings.HasPrefix(r.URL.Path, "/metrics") {
				next.ServeHTTP(w, r)
				return
			}

			normPath := normalisePath(r.URL.Path)

			// Track in-flight requests — decremented when the handler returns.
			defer m.TrackInFlight(r.Method)()

			start := time.Now()
			recorder := newMetricsResponseWriter(w)

			next.ServeHTTP(recorder, r)

			m.RecordHTTPRequest(
				r.Method,
				normPath,
				recorder.status,
				time.Since(start),
				recorder.bytesWritten,
			)
		})
	}
}

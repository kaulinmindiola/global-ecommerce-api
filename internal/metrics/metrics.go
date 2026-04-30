// Package metrics provides Prometheus instrumentation for the application.
//
// All metrics are registered on a dedicated registry (not the default global one)
// to avoid pollution from third-party packages and enable clean testing.
//
// Metric naming follows Prometheus conventions:
//   - Namespace:  "ecommerce"
//   - Subsystems: "http", "db", "cache", "business"
//   - Units:      _total (counters), _seconds (latency), _active (gauges)
package metrics

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const namespace = "ecommerce"

// Metrics holds all registered Prometheus instruments.
// One instance is created at startup and injected into the middleware and handlers.
type Metrics struct {
	registry *prometheus.Registry

	// ── HTTP metrics ───────────────────────────────────────────────────────
	// HTTPRequestsTotal counts every request partitioned by method, path and status.
	// Label cardinality is kept low: paths are normalised (/{id} not /abc-123-def).
	HTTPRequestsTotal *prometheus.CounterVec

	// HTTPRequestDuration measures end-to-end handler latency in seconds.
	// Using seconds (not ms) matches the Prometheus convention — Grafana converts.
	HTTPRequestDuration *prometheus.HistogramVec

	// HTTPRequestsInFlight measures currently active requests — useful for
	// detecting thundering herds and connection pool exhaustion.
	HTTPRequestsInFlight *prometheus.GaugeVec

	// HTTPResponseSize measures response payload sizes for bandwidth tracking.
	HTTPResponseSize *prometheus.HistogramVec

	// ── Database metrics ───────────────────────────────────────────────────
	// DBConnectionsActive is a gauge that reads directly from pgxpool stats.
	DBConnectionsActive prometheus.Gauge

	// DBConnectionsIdle reports idle connections in the pool.
	DBConnectionsIdle prometheus.Gauge

	// DBConnectionsWaiting reports requests waiting for a free connection.
	// A sustained non-zero value means the pool is too small.
	DBConnectionsWaiting prometheus.Gauge

	// DBQueryDuration measures individual query latency by operation type.
	DBQueryDuration *prometheus.HistogramVec

	// ── Cache metrics ──────────────────────────────────────────────────────
	// CacheOperationsTotal counts cache hits and misses by key type.
	// hit_ratio = hits / (hits + misses) — computed in Grafana, not here.
	CacheOperationsTotal *prometheus.CounterVec

	// ── Business metrics ───────────────────────────────────────────────────
	// OrdersCreatedTotal counts successful order creations by currency.
	OrdersCreatedTotal *prometheus.CounterVec

	// OrdersTotalRevenue tracks cumulative revenue (in USD equivalent) by currency.
	OrdersTotalRevenue *prometheus.CounterVec

	// CurrencyConversionsTotal counts conversions by currency pair.
	CurrencyConversionsTotal *prometheus.CounterVec
}

// New creates and registers all application metrics on a dedicated registry.
// The registry is returned embedded in the Metrics struct so the handler can
// serve it at GET /metrics.
func New() *Metrics {
	registry := prometheus.NewRegistry()

	// Register standard Go runtime and process metrics.
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		registry: registry,

		// ── HTTP ─────────────────────────────────────────────────────────────
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests partitioned by method, path and status code.",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request latency in seconds.",
				// Buckets cover the expected range: fast DB reads (5ms) to slow orders (2s).
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
			},
			[]string{"method", "path"},
		),

		HTTPRequestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Number of HTTP requests currently being handled.",
			},
			[]string{"method"},
		),

		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "http",
				Name:      "response_size_bytes",
				Help:      "HTTP response size in bytes.",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6), // 100B to 10MB
			},
			[]string{"method", "path"},
		),

		// ── Database ─────────────────────────────────────────────────────────
		DBConnectionsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "db",
			Name:      "connections_active",
			Help:      "Number of active PostgreSQL connections currently in use.",
		}),

		DBConnectionsIdle: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "db",
			Name:      "connections_idle",
			Help:      "Number of idle PostgreSQL connections in the pool.",
		}),

		DBConnectionsWaiting: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "db",
			Name:      "connections_waiting",
			Help:      "Number of requests waiting for a PostgreSQL connection.",
		}),

		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "db",
				Name:      "query_duration_seconds",
				Help:      "Database query latency in seconds by operation type.",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5},
			},
			[]string{"operation"}, // e.g. "create_order", "list_products"
		),

		// ── Cache ─────────────────────────────────────────────────────────────
		CacheOperationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "cache",
				Name:      "operations_total",
				Help:      "Total cache operations partitioned by result (hit/miss) and key type.",
			},
			[]string{"result", "key_type"}, // result: "hit"|"miss", key_type: "product"|"currency"|etc.
		),

		// ── Business ──────────────────────────────────────────────────────────
		OrdersCreatedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "orders_created_total",
				Help:      "Total number of orders successfully created by currency.",
			},
			[]string{"currency"},
		),

		OrdersTotalRevenue: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "revenue_usd_total",
				Help:      "Total revenue in USD equivalent by order currency.",
			},
			[]string{"currency"},
		),

		CurrencyConversionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business",
				Name:      "currency_conversions_total",
				Help:      "Total currency conversions by currency pair (from_to).",
			},
			[]string{"pair"}, // e.g. "USD_EUR"
		),
	}

	// Register all metrics.
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.HTTPResponseSize,
		m.DBConnectionsActive,
		m.DBConnectionsIdle,
		m.DBConnectionsWaiting,
		m.DBQueryDuration,
		m.CacheOperationsTotal,
		m.OrdersCreatedTotal,
		m.OrdersTotalRevenue,
		m.CurrencyConversionsTotal,
	)

	return m
}

// Registry returns the Prometheus registry for use by the /metrics handler.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// ── Instrumentation helpers ───────────────────────────────────────────────────

// RecordHTTPRequest records a completed HTTP request into all HTTP metrics.
// Called by the metrics middleware after each request finishes.
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration, responseSize int) {
	statusStr := strconv.Itoa(status)

	m.HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(responseSize))
}

// TrackInFlight increments the in-flight gauge before the request and returns
// a function that decrements it. Call with defer:
//
//	defer m.TrackInFlight(method)()
func (m *Metrics) TrackInFlight(method string) func() {
	m.HTTPRequestsInFlight.WithLabelValues(method).Inc()
	return func() {
		m.HTTPRequestsInFlight.WithLabelValues(method).Dec()
	}
}

// UpdateDBPoolStats reads current connection pool statistics from pgxpool
// and updates the corresponding gauges. Called by the health checker every minute.
func (m *Metrics) UpdateDBPoolStats(pool *pgxpool.Pool) {
	stats := pool.Stat()
	m.DBConnectionsActive.Set(float64(stats.AcquiredConns()))
	m.DBConnectionsIdle.Set(float64(stats.IdleConns()))
	m.DBConnectionsWaiting.Set(float64(stats.EmptyAcquireCount()))
}

// RecordCacheHit increments the cache hit counter for the given key type.
func (m *Metrics) RecordCacheHit(keyType string) {
	m.CacheOperationsTotal.WithLabelValues("hit", keyType).Inc()
}

// RecordCacheMiss increments the cache miss counter for the given key type.
func (m *Metrics) RecordCacheMiss(keyType string) {
	m.CacheOperationsTotal.WithLabelValues("miss", keyType).Inc()
}

// RecordOrderCreated records a successful order creation with its revenue.
func (m *Metrics) RecordOrderCreated(currency string, amountUSD float64) {
	m.OrdersCreatedTotal.WithLabelValues(currency).Inc()
	m.OrdersTotalRevenue.WithLabelValues(currency).Add(amountUSD)
}

// RecordCurrencyConversion increments the conversion counter for a currency pair.
func (m *Metrics) RecordCurrencyConversion(fromCode, toCode string) {
	pair := fromCode + "_" + toCode
	m.CurrencyConversionsTotal.WithLabelValues(pair).Inc()
}

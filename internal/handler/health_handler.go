package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/metrics"
	"github.com/redis/go-redis/v9"
)

// HealthHandler serves system observability endpoints.
//
// GET /api/v1/health   — deep dependency health check
// GET /api/v1/version  — build metadata
type HealthHandler struct {
	db         *pgxpool.Pool
	redis      *redis.Client
	metrics    *metrics.Metrics // From v1: Support for Prometheus
	version    string
	buildDate  string
	commitHash string
}

// NewHealthHandler creates a new HealthHandler.
// version, buildDate and commitHash are injected at compile time via ldflags.
func NewHealthHandler(
	db *pgxpool.Pool,
	redisClient *redis.Client,
	m *metrics.Metrics,
	version, buildDate, commitHash string,
) *HealthHandler {
	return &HealthHandler{
		db:         db,
		redis:      redisClient,
		metrics:    m,
		version:    version,
		buildDate:  buildDate,
		commitHash: commitHash,
	}
}

// ── /api/v1/health ─────────────────────────────────────────────────────────

// componentStatus represents a single dependency's health state.
// Hybrid structure: satisfies v2 strict contracts while providing v1 detailed metrics.
type componentStatus struct {
	Status    string `json:"status"`               // "up" | "down" (v2 compatibility)
	Latency   string `json:"latency,omitempty"`    // String format for v2 compatibility
	LatencyMs int64  `json:"latency_ms,omitempty"` // Int format for v1 detailed monitoring
	Message   string `json:"message,omitempty"`    // From v1

	// Database-specific fields (From v1)
	ConnectionsActive int32 `json:"connections_active,omitempty"`
	ConnectionsIdle   int32 `json:"connections_idle,omitempty"`
	MaxConns          int32 `json:"max_conns,omitempty"`
}

// healthResponse is the full payload for GET /api/v1/health.
type healthResponse struct {
	Status    string                     `json:"status"`    // "healthy" | "degraded"
	Timestamp string                     `json:"timestamp"` // UTC RFC3339
	Services  map[string]componentStatus `json:"services"`  // Kept as "services" to avoid v2 breaking changes
	Version   string                     `json:"version"`
}

// Health godoc
// @Summary      Health check
// @Description  Check if the API and its dependencies (database, redis) are operational with detailed metrics
// @Tags         Health
// @Produce      json
// @Success      200  {object}  healthResponse  "System is healthy"
// @Failure      503  {object}  healthResponse  "System is degraded or unhealthy"
// @Router       /api/v1/health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	services := make(map[string]componentStatus)
	overallHealthy := true

	// ── PostgreSQL probe ──────────────────────────────────────────────────
	dbStatus := h.probeDatabase(ctx)
	services["database"] = dbStatus
	if dbStatus.Status != "up" {
		overallHealthy = false
	}

	// ── Redis probe ───────────────────────────────────────────────────────
	redisStatus := h.probeRedis(ctx)
	services["redis"] = redisStatus
	if redisStatus.Status != "up" {
		overallHealthy = false
	}

	// Update Prometheus pool metrics while we have fresh stats. (From v1)
	if h.metrics != nil {
		h.metrics.UpdateDBPoolStats(h.db)
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !overallHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	successResponse(w, httpStatus, healthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
		Version:   h.version,
	})
}

// probeDatabase pings PostgreSQL and collects pool statistics.
func (h *HealthHandler) probeDatabase(ctx context.Context) componentStatus {
	start := time.Now()

	if err := h.db.Ping(ctx); err != nil {
		return componentStatus{
			Status:  "down",
			Message: "PostgreSQL ping failed",
		}
	}

	duration := time.Since(start)
	stats := h.db.Stat()

	return componentStatus{
		Status:            "up",
		Latency:           duration.String(),       // v2 compatible
		LatencyMs:         duration.Milliseconds(), // v1 metric
		ConnectionsActive: stats.AcquiredConns(),
		ConnectionsIdle:   stats.IdleConns(),
		MaxConns:          stats.MaxConns(),
	}
}

// probeRedis sends a PING command to Redis and measures the round-trip time.
func (h *HealthHandler) probeRedis(ctx context.Context) componentStatus {
	start := time.Now()

	if err := h.redis.Ping(ctx).Err(); err != nil {
		return componentStatus{
			Status:  "down",
			Message: "Redis ping failed",
		}
	}

	duration := time.Since(start)

	return componentStatus{
		Status:    "up",
		Latency:   duration.String(),       // v2 compatible
		LatencyMs: duration.Milliseconds(), // v1 metric
	}
}

// ── /api/v1/version ────────────────────────────────────────────────────────

// versionResponse is the payload for GET /api/v1/version.
type versionResponse struct {
	Version    string `json:"version"`
	BuildDate  string `json:"build_date"`
	CommitHash string `json:"commit_hash"`
	GoVersion  string `json:"go_version"`
}

// Version godoc
// @Summary      Get API version
// @Description  Returns API version, build date and commit hash
// @Tags         Health
// @Produce      json
// @Success      200  {object}  versionResponse
// @Router       /api/v1/version [get]
func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	successResponse(w, http.StatusOK, versionResponse{
		Version:    h.version,
		BuildDate:  h.buildDate,
		CommitHash: h.commitHash,
	})
}

package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthHandler handles system observability endpoints.
// GET /api/v1/health
// GET /api/v1/version
type HealthHandler struct {
	db         *pgxpool.Pool
	redis      *redis.Client
	version    string
	buildDate  string
	commitHash string
}

// NewHealthHandler creates a new HealthHandler.
// version, buildDate, and commitHash are injected at build time via ldflags.
func NewHealthHandler(db *pgxpool.Pool, redisClient *redis.Client, version, buildDate, commitHash string) *HealthHandler {
	return &HealthHandler{
		db:         db,
		redis:      redisClient,
		version:    version,
		buildDate:  buildDate,
		commitHash: commitHash,
	}
}

// ─────────────────────────────────────────────
// GET /api/v1/health
// ─────────────────────────────────────────────

// serviceStatus holds the up/down status of a single dependency.
type serviceStatus struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
}

// healthResponse matches the spec contract on page 1.
type healthResponse struct {
	Status    string                   `json:"status"`
	Timestamp string                   `json:"timestamp"`
	Services  map[string]serviceStatus `json:"services"`
	Version   string                   `json:"version"`
}

// Health probes both PostgreSQL and Redis and returns their status.
// Response: 200 OK when all services are healthy, 503 if any dependency is down.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	services := make(map[string]serviceStatus)
	overallHealthy := true

	// Probe PostgreSQL.
	dbStart := time.Now()
	if err := h.db.Ping(ctx); err != nil {
		services["database"] = serviceStatus{Status: "down"}
		overallHealthy = false
	} else {
		services["database"] = serviceStatus{
			Status:  "up",
			Latency: time.Since(dbStart).String(),
		}
	}

	// Probe Redis.
	redisStart := time.Now()
	if err := h.redis.Ping(ctx).Err(); err != nil {
		services["redis"] = serviceStatus{Status: "down"}
		overallHealthy = false
	} else {
		services["redis"] = serviceStatus{
			Status:  "up",
			Latency: time.Since(redisStart).String(),
		}
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

// ─────────────────────────────────────────────
// GET /api/v1/version
// ─────────────────────────────────────────────

// versionResponse matches the spec contract on page 2.
type versionResponse struct {
	Version    string `json:"version"`
	BuildDate  string `json:"build_date"`
	GoVersion  string `json:"go_version"`
	CommitHash string `json:"commit_hash"`
}

// Version returns build metadata for the running server instance.
// Response: 200 OK
func (h *HealthHandler) Version(w http.ResponseWriter, r *http.Request) {
	successResponse(w, http.StatusOK, versionResponse{
		Version:    h.version,
		BuildDate:  h.buildDate,
		CommitHash: h.commitHash,
	})
}

package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
)

// ─────────────────────────────────────────────
// Standard response envelopes
// ─────────────────────────────────────────────

// successResponse writes a JSON payload with the given HTTP status code.
// All successful responses are wrapped in a consistent envelope.
func successResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// noContentResponse writes a 204 No Content with no body.
func noContentResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────
// Error envelope — matches spec page 3 exactly
// ─────────────────────────────────────────────

// ErrorResponse is the top-level error envelope sent to clients.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains the machine-readable error code, human message,
// optional field-level details, and request metadata for log correlation.
type ErrorBody struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	Details   []ErrorDetail `json:"details,omitempty"`
	Timestamp string        `json:"timestamp"`
	Path      string        `json:"path"`
	RequestID string        `json:"request_id"`
}

// ErrorDetail represents a single field-level validation failure.
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// errorResponse writes a structured error JSON response.
// path and requestID are extracted from the request context by the middleware layer.
func errorResponse(w http.ResponseWriter, r *http.Request, status int, code, message string, details ...ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	body := ErrorBody{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      r.URL.Path,
		RequestID: requestIDFromContext(r.Context()),
	}

	if len(details) > 0 {
		body.Details = details
	}

	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: body})
}

// ─────────────────────────────────────────────
// Domain error → HTTP status mapper
// ─────────────────────────────────────────────

// domainErrorResponse maps a domain sentinel error to the correct HTTP status
// code and machine-readable error code string defined in the spec.
// This is the single place in the codebase where domain errors become HTTP responses.
func domainErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		errorResponse(w, r, http.StatusNotFound,
			"RESOURCE_NOT_FOUND", "The requested resource does not exist")

	case errors.Is(err, domain.ErrConflict):
		errorResponse(w, r, http.StatusConflict,
			"RESOURCE_CONFLICT", "A resource with the same identifier already exists")

	case errors.Is(err, domain.ErrInvalidInput):
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", err.Error())

	case errors.Is(err, domain.ErrInvalidReference):
		errorResponse(w, r, http.StatusUnprocessableEntity,
			"INVALID_REFERENCE", "A referenced resource does not exist")

	case errors.Is(err, domain.ErrInsufficientStock):
		errorResponse(w, r, http.StatusUnprocessableEntity,
			"BUSINESS_INSUFFICIENT_STOCK", "One or more items do not have sufficient stock")

	case errors.Is(err, domain.ErrUnauthorized):
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_INVALID_CREDENTIALS", "Invalid email or password")

	case errors.Is(err, domain.ErrForbidden):
		errorResponse(w, r, http.StatusForbidden,
			"AUTH_INSUFFICIENT_PERMISSIONS", "You do not have permission to access this resource")

	case errors.Is(err, domain.ErrInvalidTransition):
		errorResponse(w, r, http.StatusUnprocessableEntity,
			"BUSINESS_INVALID_OPERATION", "This status transition is not allowed")

	default:
		errorResponse(w, r, http.StatusInternalServerError,
			"SYSTEM_INTERNAL_ERROR", "An unexpected error occurred")
	}
}

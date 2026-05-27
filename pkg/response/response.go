// Package response provides helpers for writing consistent JSON responses
// across all HTTP handlers.
//
// Every successful response and every error response in the API goes through
// this package — it is the single place that controls the response envelope shape.
//
// Usage:
//
//	response.JSON(w, http.StatusOK, data)
//	response.Created(w, newProduct)
//	response.NoContent(w)
//	response.Error(w, r, appErrors.NewNotFoundError("Product", id))
//	response.Paginated(w, items, total, page, pageSize)
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	apperrors "github.com/kaulinmindiola/global-ecommerce-api/pkg/errors"
)

// contextRequestIDKey matches the key used by the middleware package.
// Defined here to avoid importing middleware (circular dependency).
type contextKey string

const contextKeyRequestID contextKey = "request_id"

// ── Success responses ────────────────────────────────────────────────────────

// JSON writes a JSON-encoded payload with the given HTTP status code.
// It sets Content-Type: application/json and encodes data in one operation.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// OK writes a 200 OK JSON response.
func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

// Created writes a 201 Created JSON response.
// Used after successfully creating a new resource.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

// NoContent writes a 204 No Content response with no body.
// Used for successful DELETE operations and logout.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// ── Paginated response ───────────────────────────────────────────────────────

// PaginationMeta holds the metadata included in every paginated response.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// PaginatedResponse is the envelope for paginated list endpoints.
type PaginatedResponse struct {
	Data       any            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// Paginated writes a 200 OK response with data and pagination metadata.
// HasNext and HasPrev are computed automatically from page and total.
//
// Example:
//
//	response.Paginated(w, products, 150, 2, 20)
//	// → {"data":[...], "pagination":{"page":2,"limit":20,"total":150,"total_pages":8,...}}
func Paginated(w http.ResponseWriter, data any, total int64, page, pageSize int) {
	if pageSize <= 0 {
		pageSize = 20
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	JSON(w, http.StatusOK, PaginatedResponse{
		Data: data,
		Pagination: PaginationMeta{
			Page:       page,
			Limit:      pageSize,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	})
}

// ── Error responses ──────────────────────────────────────────────────────────

// ErrorEnvelope is the top-level JSON structure for all error responses.
// Matches the spec defined in HTTP_Status_Codes__Error_Handling.pdf page 3.
type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains all fields of the error response payload.
type ErrorBody struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Details   []apperrors.FieldError `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Path      string                 `json:"path"`
	RequestID string                 `json:"request_id"`
}

// Error writes a structured JSON error response derived from an AppError.
// This is THE function all handlers call when they need to return an error.
//
// Example:
//
//	response.Error(w, r, apperrors.ErrNotFound)
//	response.Error(w, r, apperrors.NewValidationError("Invalid input",
//	    apperrors.FieldError{Field: "email", Message: "Invalid format", Code: "INVALID_FORMAT"},
//	))
func Error(w http.ResponseWriter, r *http.Request, err *apperrors.AppError) {
	requestID, ok := r.Context().Value(contextKeyRequestID).(string)
	if !ok {
		requestID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)

	envelope := ErrorEnvelope{
		Error: ErrorBody{
			Code:      err.Code,
			Message:   err.Message,
			Details:   err.Details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Path:      r.URL.Path,
			RequestID: requestID,
		},
	}

	if encErr := json.NewEncoder(w).Encode(envelope); encErr != nil {
		slog.Error("failed to encode error response", "error", encErr)
	}
}

// ErrorFromDomain converts a domain/service error to an AppError and writes
// the appropriate HTTP response. This is the bridge between the service layer
// and the HTTP layer — handlers call this instead of doing their own error mapping.
//
// Example:
//
//	product, err := h.productService.GetByID(ctx, id)
//	if err != nil {
//	    response.ErrorFromDomain(w, r, err)
//	    return
//	}
func ErrorFromDomain(w http.ResponseWriter, r *http.Request, err error) {
	appErr := apperrors.FromDomainError(err)
	Error(w, r, appErr)
}

// ── Convenience status writers ───────────────────────────────────────────────

// BadRequest writes a 400 with a validation error message.
func BadRequest(w http.ResponseWriter, r *http.Request, message string, details ...apperrors.FieldError) {
	Error(w, r, apperrors.NewValidationError(message, details...))
}

// NotFound writes a 404 for a specific resource.
func NotFound(w http.ResponseWriter, r *http.Request, resource string, id any) {
	Error(w, r, apperrors.NewNotFoundError(resource, id))
}

// Unauthorized writes a 401 for missing or invalid authentication.
func Unauthorized(w http.ResponseWriter, r *http.Request) {
	Error(w, r, apperrors.ErrTokenMissing)
}

// Forbidden writes a 403 for insufficient permissions.
func Forbidden(w http.ResponseWriter, r *http.Request) {
	Error(w, r, apperrors.ErrForbidden)
}

// InternalServerError writes a 500 and logs the underlying cause.
func InternalServerError(w http.ResponseWriter, r *http.Request, cause error) {
	slog.Error("internal server error",
		"error", cause,
		"path", r.URL.Path,
		"method", r.Method,
	)
	Error(w, r, apperrors.NewInternalError(cause))
}

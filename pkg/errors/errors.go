// Package errors defines application-level error types that carry both
// machine-readable codes and HTTP status information.
//
// Design goals:
//   - Every error the API can return has an explicit type and code.
//   - The HTTP layer maps error types to status codes in one place.
//   - Error messages are safe to expose to clients (no internal stack traces).
//   - Errors can be wrapped with fmt.Errorf("%w") and still detected via errors.As.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ── Error code constants ─────────────────────────────────────────────────────
// Machine-readable identifiers sent in the API error envelope.
// Clients use these codes to handle specific error conditions programmatically.

const (
	// 400 Bad Request
	CodeValidationError = "VALIDATION_ERROR"
	CodeInvalidFormat   = "INVALID_FORMAT"
	CodeMissingField    = "MISSING_REQUIRED_FIELD"

	// 401 Unauthorized
	CodeTokenMissing       = "AUTH_TOKEN_MISSING"
	CodeTokenInvalid       = "AUTH_TOKEN_INVALID"
	CodeTokenExpired       = "AUTH_TOKEN_EXPIRED"
	CodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"

	// 403 Forbidden
	CodeForbidden = "AUTH_INSUFFICIENT_PERMISSIONS"

	// 404 Not Found
	CodeNotFound = "RESOURCE_NOT_FOUND"

	// 409 Conflict
	CodeConflict = "RESOURCE_CONFLICT"

	// 422 Unprocessable Entity
	CodeInvalidReference  = "INVALID_REFERENCE"
	CodeInsufficientStock = "BUSINESS_INSUFFICIENT_STOCK"
	CodeInvalidCurrency   = "BUSINESS_INVALID_CURRENCY"
	CodeInvalidOperation  = "BUSINESS_INVALID_OPERATION"
	CodeCalculationError  = "BUSINESS_CALCULATION_ERROR"

	// 500 Internal Server Error
	CodeInternalError = "SYSTEM_INTERNAL_ERROR"

	// 503 Service Unavailable
	CodeDatabaseError    = "SYSTEM_DATABASE_ERROR"
	CodeCacheError       = "SYSTEM_CACHE_ERROR"
	CodeExternalAPIError = "SYSTEM_EXTERNAL_API_ERROR"
)

// ── Base AppError type ───────────────────────────────────────────────────────

// AppError is the base error type for all application-level errors.
// It implements the error interface and carries HTTP metadata.
type AppError struct {
	// Code is the machine-readable error identifier (see constants above).
	Code string `json:"code"`
	// Message is a human-readable description safe for client exposure.
	Message string `json:"message"`
	// HTTPStatus is the HTTP status code this error maps to.
	HTTPStatus int `json:"-"`
	// Details holds optional field-level validation errors.
	Details []FieldError `json:"details,omitempty"`
	// cause is the underlying error for internal logging (never sent to client).
	cause error
}

// FieldError represents a single field-level validation failure.
// Matches the error detail structure defined in the HTTP spec.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

// Unwrap allows errors.Is and errors.As to inspect the cause chain.
func (e *AppError) Unwrap() error {
	return e.cause
}

// WithCause attaches an underlying error for internal logging.
// The cause is never serialised to JSON — it is for server-side diagnostics only.
func (e *AppError) WithCause(err error) *AppError {
	clone := *e
	clone.cause = err
	return &clone
}

// WithDetails attaches field-level validation errors to the AppError.
func (e *AppError) WithDetails(details ...FieldError) *AppError {
	clone := *e
	clone.Details = append(clone.Details, details...)
	return &clone
}

// Is enables errors.Is comparisons by matching error codes.
// This lets callers do: errors.Is(err, errors.ErrNotFound)
func (e *AppError) Is(target error) bool {
	var t *AppError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// ── Sentinel errors ──────────────────────────────────────────────────────────
// Pre-built AppError instances for the most common error cases.
// Use these directly or call their constructors for custom messages.

var (
	ErrNotFound = &AppError{
		Code:       CodeNotFound,
		Message:    "The requested resource does not exist",
		HTTPStatus: http.StatusNotFound,
	}

	ErrConflict = &AppError{
		Code:       CodeConflict,
		Message:    "A resource with the same identifier already exists",
		HTTPStatus: http.StatusConflict,
	}

	ErrUnauthorized = &AppError{
		Code:       CodeInvalidCredentials,
		Message:    "Invalid credentials",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrTokenMissing = &AppError{
		Code:       CodeTokenMissing,
		Message:    "Authorization header is required",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrTokenInvalid = &AppError{
		Code:       CodeTokenInvalid,
		Message:    "Token is invalid or has expired",
		HTTPStatus: http.StatusUnauthorized,
	}

	ErrForbidden = &AppError{
		Code:       CodeForbidden,
		Message:    "You do not have permission to access this resource",
		HTTPStatus: http.StatusForbidden,
	}

	ErrInsufficientStock = &AppError{
		Code:       CodeInsufficientStock,
		Message:    "One or more items do not have sufficient stock",
		HTTPStatus: http.StatusUnprocessableEntity,
	}

	ErrInvalidOperation = &AppError{
		Code:       CodeInvalidOperation,
		Message:    "This operation is not allowed in the current state",
		HTTPStatus: http.StatusUnprocessableEntity,
	}

	ErrInternal = &AppError{
		Code:       CodeInternalError,
		Message:    "An unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
	}
)

// ── Constructor functions ────────────────────────────────────────────────────

// NewValidationError creates a 400 validation error with optional field details.
//
// Example:
//
//	return errors.NewValidationError("Request validation failed",
//	    errors.FieldError{Field: "email", Message: "Invalid format", Code: "INVALID_FORMAT"},
//	    errors.FieldError{Field: "password", Message: "Too short", Code: "TOO_SHORT"},
//	)
func NewValidationError(message string, details ...FieldError) *AppError {
	return &AppError{
		Code:       CodeValidationError,
		Message:    message,
		HTTPStatus: http.StatusBadRequest,
		Details:    details,
	}
}

// NewNotFoundError creates a 404 error for a specific resource type.
//
// Example:
//
//	return errors.NewNotFoundError("Product", productID)
func NewNotFoundError(resource string, id any) *AppError {
	return &AppError{
		Code:       CodeNotFound,
		Message:    fmt.Sprintf("%s with ID %v was not found", resource, id),
		HTTPStatus: http.StatusNotFound,
	}
}

// NewConflictError creates a 409 error for uniqueness violations.
//
// Example:
//
//	return errors.NewConflictError("User", "email", req.Email)
func NewConflictError(resource, field string, value any) *AppError {
	return &AppError{
		Code:       CodeConflict,
		Message:    fmt.Sprintf("%s with %s '%v' already exists", resource, field, value),
		HTTPStatus: http.StatusConflict,
	}
}

// NewBusinessError creates a 422 error for business rule violations.
func NewBusinessError(code, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

// NewInternalError creates a 500 error, wrapping an internal cause for logging.
// The cause is never exposed to clients.
func NewInternalError(cause error) *AppError {
	return &AppError{
		Code:       CodeInternalError,
		Message:    "An unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
		cause:      cause,
	}
}

// ── Domain error bridge ──────────────────────────────────────────────────────

// FromDomainError converts a domain sentinel error to an AppError.
// This is the single translation point between the domain and HTTP layers.
// Called by the response package's Error function.
func FromDomainError(err error) *AppError {
	// Check if it is already an AppError (from service/handler layer).
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Map domain sentinel errors to AppErrors.
	switch {
	case err.Error() == "resource not found":
		return ErrNotFound
	case err.Error() == "resource already exists":
		return ErrConflict
	case err.Error() == "authentication required":
		return ErrUnauthorized
	case err.Error() == "access denied":
		return ErrForbidden
	case err.Error() == "insufficient stock":
		return ErrInsufficientStock
	case err.Error() == "invalid status transition":
		return ErrInvalidOperation
	default:
		return NewInternalError(err)
	}
}

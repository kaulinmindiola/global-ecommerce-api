package domain

import "errors"

// Sentinel errors used across the service and repository layers.
// Handlers map these to their corresponding HTTP status codes.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	// Maps to HTTP 404 Not Found.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict is returned when a unique constraint is violated (e.g., duplicate email or SKU).
	// Maps to HTTP 409 Conflict.
	ErrConflict = errors.New("resource already exists")

	// ErrInvalidInput is returned when request data fails business rule validation.
	// Maps to HTTP 400 Bad Request.
	ErrInvalidInput = errors.New("invalid input")

	// ErrInvalidReference is returned when a foreign key references a non-existent record.
	// Maps to HTTP 422 Unprocessable Entity.
	ErrInvalidReference = errors.New("invalid reference to related resource")

	// ErrInsufficientStock is returned when an order requests more units than available.
	// Maps to HTTP 422 Unprocessable Entity.
	ErrInsufficientStock = errors.New("insufficient stock")

	// ErrUnauthorized is returned when authentication credentials are missing or invalid.
	// Maps to HTTP 401 Unauthorized.
	ErrUnauthorized = errors.New("authentication required")

	// ErrForbidden is returned when an authenticated user attempts to access a resource
	// they do not own or have permission to modify.
	// Maps to HTTP 403 Forbidden.
	ErrForbidden = errors.New("access denied")

	// ErrInvalidTransition is returned when an order status change violates the state machine.
	// Maps to HTTP 422 Unprocessable Entity.
	ErrInvalidTransition = errors.New("invalid status transition")
)

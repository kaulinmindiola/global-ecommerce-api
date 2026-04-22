package domain

import "errors"

// Sentinel errors for the domain layer.
// These allow the repository to translate infrastructure-specific errors
// (like Postgres codes) into business-friendly errors.
var (
	ErrNotFound          = errors.New("resource not found")
	ErrConflict          = errors.New("resource already exists or conflicts with another")
	ErrInvalidReference  = errors.New("invalid reference to a related resource")
	ErrInvalidInput      = errors.New("invalid input data")
	ErrInsufficientStock = errors.New("insufficient stock for product")
)

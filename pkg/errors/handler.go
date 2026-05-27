package errors

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// contextKey matches the key used by the middleware package to store request IDs.
// Redeclared here to avoid a circular import (middleware → errors → middleware).
type contextKey string

const contextKeyRequestID contextKey = "request_id"

// ── HTTP error writer ─────────────────────────────────────────────────────────

// Write serialises an AppError into the standard error envelope and sends it
// as a JSON HTTP response. This is the single function responsible for turning
// an AppError into bytes on the wire.
//
// It is intentionally low-level — the response package builds on top of this.
// Handlers should use response.Error() or response.ErrorFromDomain() instead of
// calling this directly.
func Write(w http.ResponseWriter, r *http.Request, err *AppError) {
	requestID, ok := r.Context().Value(contextKeyRequestID).(string)
	if !ok {
		requestID = "unknown"
	}

	envelope := errorEnvelope{
		Error: errorBody{
			Code:      err.Code,
			Message:   err.Message,
			Details:   err.Details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Path:      r.URL.Path,
			RequestID: requestID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)

	if encErr := json.NewEncoder(w).Encode(envelope); encErr != nil {
		slog.Error("failed to encode error response",
			"error", encErr,
			"original_code", err.Code,
		)
	}
}

// WriteDomain converts a domain/service error to an AppError and writes the
// HTTP response. This is the convenience method used by most handlers:
//
//	if err != nil {
//	    errors.WriteDomain(w, r, err)
//	    return
//	}
func WriteDomain(w http.ResponseWriter, r *http.Request, err error) {
	Write(w, r, FromDomainError(err))
}

// WriteValidation constructs a 400 validation error with field details and
// writes it immediately. Used when request parsing fails.
//
//	errors.WriteValidation(w, r, "Invalid request body",
//	    errors.FieldError{Field: "email", Message: "Invalid format", Code: "INVALID_FORMAT"},
//	)
func WriteValidation(w http.ResponseWriter, r *http.Request, message string, details ...FieldError) {
	Write(w, r, NewValidationError(message, details...))
}

// ── JSON envelope types ───────────────────────────────────────────────────────
// Defined here (not in errors.go) to keep serialisation logic co-located with
// the Write function. The AppError struct itself never has json tags for the
// HTTP envelope — those are defined here for clean separation.

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Details   []FieldError `json:"details,omitempty"`
	Timestamp string       `json:"timestamp"`
	Path      string       `json:"path"`
	RequestID string       `json:"request_id"`
}

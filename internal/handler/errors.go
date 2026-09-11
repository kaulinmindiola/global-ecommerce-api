package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Esta función DEBE estar aquí para que todos los handlers la usen
func parseUUIDParam(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	val := chi.URLParam(r, param)
	id, err := uuid.Parse(val)
	if err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid ID format — must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

// Asegúrate de tener también aquí errorResponse, successResponse, etc.

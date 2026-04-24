package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// UserHandler handles all user profile HTTP endpoints.
// GET    /api/v1/users/me
// PUT    /api/v1/users/me
// DELETE /api/v1/users/me
// GET    /api/v1/users/{id}  (admin)
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler creates a new UserHandler with required dependencies.
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// ─────────────────────────────────────────────
// GET /api/v1/users/me
// ─────────────────────────────────────────────

// GetMe returns the authenticated user's profile.
// Response: 200 OK
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	profile, err := h.userService.GetProfile(r.Context(), userID)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, profile)
}

// ─────────────────────────────────────────────
// PUT /api/v1/users/me
// ─────────────────────────────────────────────

// updateMeRequest is the partial-update JSON body for profile changes.
// All fields are optional — only non-empty values are applied.
type updateMeRequest struct {
	FullName          string `json:"full_name"`
	PreferredCurrency string `json:"preferred_currency"`
	PreferredTimezone string `json:"preferred_timezone"`
}

// UpdateMe modifies the authenticated user's profile.
// Response: 200 OK
func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	var req updateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	updated, err := h.userService.UpdateProfile(r.Context(), userID, service.UpdateProfileRequest{
		FullName:          req.FullName,
		PreferredCurrency: req.PreferredCurrency,
		PreferredTimezone: req.PreferredTimezone,
	})
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, updated)
}

// ─────────────────────────────────────────────
// DELETE /api/v1/users/me
// ─────────────────────────────────────────────

// DeleteMe performs a soft-delete on the authenticated user's account.
// Response: 204 No Content
func (h *UserHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	if err := h.userService.Delete(r.Context(), userID); err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	noContentResponse(w)
}

// ─────────────────────────────────────────────
// GET /api/v1/users/{id}
// ─────────────────────────────────────────────

// GetByID retrieves any user's profile by UUID (admin endpoint).
// Response: 200 OK
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid user ID format — must be a valid UUID")
		return
	}

	profile, err := h.userService.GetProfile(r.Context(), userID)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, profile)
}

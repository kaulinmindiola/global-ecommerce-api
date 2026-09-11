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

// GetMe godoc
// @Summary      Get current user profile
// @Description  Retrieve the authenticated user's profile information based on their JWT token
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  service.UserResponse  "User profile"
// @Failure      401  {object}  ErrorResponse         "Unauthorized - Invalid or missing token"
// @Failure      404  {object}  ErrorResponse         "User not found"
// @Failure      500  {object}  ErrorResponse         "Internal Server Error"
// @Router       /users/me [get]
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
	FullName          string `json:"full_name" example:"John Doe"`
	PreferredCurrency string `json:"preferred_currency" example:"EUR"`
	PreferredTimezone string `json:"preferred_timezone" example:"Europe/Madrid"`
}

// UpdateMe godoc
// @Summary      Update user profile
// @Description  Update authenticated user's profile information. All fields are optional (partial update).
// @Tags         Users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      handler.updateMeRequest  true  "Fields to update"
// @Success      200      {object}  service.UserResponse     "User updated successfully"
// @Failure      400      {object}  ErrorResponse            "Validation Error (e.g., Invalid JSON or Currency)"
// @Failure      401      {object}  ErrorResponse            "Unauthorized"
// @Failure      500      {object}  ErrorResponse            "Internal Server Error"
// @Router       /users/me [put]
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

// DeleteMe godoc
// @Summary      Delete user account
// @Description  Perform a soft-delete on the authenticated user's account
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Success      204  "Account deleted successfully (No Content)"
// @Failure      401  {object}  ErrorResponse  "Unauthorized"
// @Failure      500  {object}  ErrorResponse  "Internal Server Error"
// @Router       /users/me [delete]
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

// GetByID godoc
// @Summary      Get user profile by ID (Admin)
// @Description  Retrieve any user's profile by their UUID. Intended for administrative use.
// @Tags         Users
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string                true  "User UUID" format(uuid)
// @Success      200  {object}  service.UserResponse  "User profile"
// @Failure      400  {object}  ErrorResponse         "Validation Error (Invalid UUID format)"
// @Failure      401  {object}  ErrorResponse         "Unauthorized"
// @Failure      404  {object}  ErrorResponse         "User not found"
// @Failure      500  {object}  ErrorResponse         "Internal Server Error"
// @Router       /users/{id} [get]
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

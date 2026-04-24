package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// AuthHandler handles all authentication-related HTTP endpoints.
// POST /api/v1/auth/register
// POST /api/v1/auth/login
// POST /api/v1/auth/logout
// POST /api/v1/auth/refresh
type AuthHandler struct {
	userService service.UserService
	authService service.AuthService
}

// NewAuthHandler creates a new AuthHandler with required dependencies.
func NewAuthHandler(userService service.UserService, authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		authService: authService,
	}
}

// ─────────────────────────────────────────────
// POST /api/v1/auth/register
// ─────────────────────────────────────────────

// registerRequest is the JSON body for user registration.
type registerRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	FullName          string `json:"full_name"`
	PreferredCurrency string `json:"preferred_currency"`
	PreferredTimezone string `json:"preferred_timezone"`
}

// Register creates a new user account and returns a JWT token pair.
// Response: 201 Created
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	user, err := h.userService.Register(r.Context(), service.RegisterRequest{
		Email:             req.Email,
		Password:          req.Password,
		FullName:          req.FullName,
		PreferredCurrency: req.PreferredCurrency,
		PreferredTimezone: req.PreferredTimezone,
	})
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	// Issue JWT tokens immediately after registration so the user
	// does not need to call /login as a second step.
	tokens, err := h.authService.IssueTokens(r.Context(), user.ID)
	if err != nil {
		errorResponse(w, r, http.StatusInternalServerError,
			"SYSTEM_INTERNAL_ERROR", "Failed to issue authentication tokens")
		return
	}

	successResponse(w, http.StatusCreated, map[string]any{
		"user":  user,
		"token": tokens,
	})
}

// ─────────────────────────────────────────────
// POST /api/v1/auth/login
// ─────────────────────────────────────────────

// loginRequest is the JSON body for user authentication.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login authenticates a user and returns a JWT access token.
// Response: 200 OK
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Email and password are required")
		return
	}

	// VerifyCredentials returns ErrUnauthorized with a generic message
	// to prevent user enumeration attacks.
	user, err := h.userService.VerifyCredentials(r.Context(), req.Email, req.Password)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	tokens, err := h.authService.IssueTokens(r.Context(), user.ID)
	if err != nil {
		errorResponse(w, r, http.StatusInternalServerError,
			"SYSTEM_INTERNAL_ERROR", "Failed to issue authentication tokens")
		return
	}

	successResponse(w, http.StatusOK, map[string]any{
		"token": tokens,
	})
}

// ─────────────────────────────────────────────
// POST /api/v1/auth/logout
// ─────────────────────────────────────────────

// Logout invalidates the current JWT by adding it to a blocklist in Redis.
// Response: 204 No Content
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract the raw token from the Authorization header.
	rawToken := extractBearerToken(r)
	if rawToken == "" {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authorization header is required")
		return
	}

	if err := h.authService.RevokeToken(r.Context(), rawToken); err != nil {
		errorResponse(w, r, http.StatusInternalServerError,
			"SYSTEM_INTERNAL_ERROR", "Failed to revoke token")
		return
	}

	noContentResponse(w)
}

// ─────────────────────────────────────────────
// POST /api/v1/auth/refresh
// ─────────────────────────────────────────────

// refreshRequest holds the refresh token for token rotation.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken exchanges a valid refresh token for a new access token pair.
// Response: 200 OK
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "refresh_token is required")
		return
	}

	tokens, err := h.authService.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_INVALID", "Refresh token is invalid or expired")
		return
	}

	successResponse(w, http.StatusOK, map[string]any{
		"token": tokens,
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// extractBearerToken parses the raw JWT from the Authorization header.
// Returns an empty string if the header is missing or malformed.
func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// TokenPair is the response shape for all token-issuing endpoints.
// Matches the spec contract on page 3.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"` // seconds
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

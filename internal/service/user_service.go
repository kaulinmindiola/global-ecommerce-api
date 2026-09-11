package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// UserService defines all business operations related to user management.
type UserService interface {
	// Register creates a new user account after validating all business rules.
	Register(ctx context.Context, req RegisterRequest) (*UserResponse, error)

	// GetProfile retrieves a user's public profile by ID.
	GetProfile(ctx context.Context, id uuid.UUID) (*UserResponse, error)

	// UpdateProfile modifies mutable profile fields for an authenticated user.
	UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*UserResponse, error)

	// VerifyCredentials checks email + password and returns the domain user on success.
	// Used internally by AuthService during login.
	VerifyCredentials(ctx context.Context, email, password string) (*domain.User, error)

	// Delete performs a soft delete on a user account.
	Delete(ctx context.Context, id uuid.UUID) error
}

// --- Request / Response DTOs ---

// RegisterRequest is the validated input for user registration.
// Field requirements match the API contract spec (page 2-3).
type RegisterRequest struct {
	Email             string
	Password          string
	FullName          string
	PreferredCurrency string // ISO 4217 code (e.g. "USD")
	PreferredTimezone string // IANA tz string (e.g. "America/Bogota")
}

// UpdateProfileRequest contains fields the user may change post-registration.
type UpdateProfileRequest struct {
	FullName          string
	PreferredCurrency string
	PreferredTimezone string
}

// UserResponse is the public-safe representation of a user.
// Critically: PasswordHash is NEVER included in this struct.
type UserResponse struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	FullName          string    `json:"full_name"`
	PreferredCurrency string    `json:"preferred_currency"`
	PreferredTimezone string    `json:"preferred_timezone"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

// userService is the concrete implementation of UserService.
type userService struct {
	userRepo     repository.UserRepository
	currencyRepo repository.CurrencyRepository
	cache        repository.CacheRepository
	currency     CurrencyService

	// bcryptCost controls password hashing cost.
	// bcrypt.DefaultCost (10) is the safe minimum for production.
	bcryptCost int
}

// NewUserService constructs a UserService with required dependencies.
func NewUserService(
	userRepo repository.UserRepository,
	currencyRepo repository.CurrencyRepository,
	cache repository.CacheRepository,
	currency CurrencyService,
) UserService {
	return &userService{
		userRepo:     userRepo,
		currencyRepo: currencyRepo,
		cache:        cache,
		currency:     currency,
		bcryptCost:   bcrypt.DefaultCost,
	}
}

// Register creates a new user account.
//
// Business rules enforced:
//   - Email must be unique across the system
//   - Password: min 8 chars, 1 uppercase, 1 digit, 1 special character
//   - PreferredCurrency must be an active ISO 4217 code in the system
//   - PreferredTimezone must be a valid IANA timezone string
//   - Password is hashed with bcrypt before persistence — NEVER stored in plain text
func (s *userService) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	if err := validateRegisterRequest(req); err != nil {
		return nil, err
	}

	// Normalize email to lowercase for consistent uniqueness checks.
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Resolve preferred currency — validates the code exists in our system.
	currency, err := s.currency.GetByCode(ctx, req.PreferredCurrency)
	if err != nil {
		return nil, fmt.Errorf("invalid preferred currency %q: %w", req.PreferredCurrency, domain.ErrInvalidInput)
	}

	// Validate IANA timezone before persisting.
	if err := validateTimezone(req.PreferredTimezone); err != nil {
		return nil, err
	}

	// Hash password. bcrypt automatically generates a cryptographic salt.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:                  uuid.New(),
		Email:               email,
		FullName:            strings.TrimSpace(req.FullName),
		PasswordHash:        string(hashedPassword),
		PreferredCurrencyID: currency.ID,
		PreferredTimezone:   req.PreferredTimezone,
		IsActive:            true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return toUserResponse(user, currency.Code), nil
}

// GetProfile retrieves a user's public profile with caching.
// Cache key: "user:{id}:profile"
func (s *userService) GetProfile(ctx context.Context, id uuid.UUID) (*UserResponse, error) {
	cacheKey := fmt.Sprintf("user:%s:profile", id)

	var cached UserResponse
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching user profile %s: %w", id, err)
	}

	// Resolve currency code for the response.
	currency, err := s.currencyRepo.GetByID(ctx, user.PreferredCurrencyID)
	if err != nil {
		return nil, fmt.Errorf("resolving preferred currency for user %s: %w", id, err)
	}

	response := toUserResponse(user, currency.Code)
	_ = s.cache.Set(ctx, cacheKey, response, 15*time.Minute)

	return response, nil
}

// UpdateProfile applies partial changes to a user's profile.
// Only non-empty fields in the request overwrite existing values.
func (s *userService) UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching user %s for update: %w", id, err)
	}

	var currencyCode string

	// Resolve the current currency code for the response.
	currentCurrency, err := s.currencyRepo.GetByID(ctx, user.PreferredCurrencyID)
	if err != nil {
		return nil, fmt.Errorf("resolving current currency: %w", err)
	}
	currencyCode = currentCurrency.Code

	// Apply partial updates.
	if req.FullName != "" {
		user.FullName = strings.TrimSpace(req.FullName)
	}

	if req.PreferredCurrency != "" {
		currency, err := s.currency.GetByCode(ctx, req.PreferredCurrency)
		if err != nil {
			return nil, fmt.Errorf("invalid preferred currency %q: %w", req.PreferredCurrency, domain.ErrInvalidInput)
		}
		user.PreferredCurrencyID = currency.ID
		currencyCode = currency.Code
	}

	if req.PreferredTimezone != "" {
		if err := validateTimezone(req.PreferredTimezone); err != nil {
			return nil, err
		}
		user.PreferredTimezone = req.PreferredTimezone
	}

	user.UpdatedAt = time.Now().UTC()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("persisting user update %s: %w", id, err)
	}

	// Invalidate profile cache.
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s:profile", id))

	return toUserResponse(user, currencyCode), nil
}

// VerifyCredentials validates email + password credentials.
// Returns domain.ErrUnauthorized if either credential is wrong.
// The error message is intentionally generic to prevent user enumeration attacks.
func (s *userService) VerifyCredentials(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		// Return a generic error regardless of whether the user was found or not.
		return nil, domain.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	return user, nil
}

// Delete soft-deletes a user account and clears their cache entries.
func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting user %s: %w", id, err)
	}

	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s:profile", id))

	return nil
}

// toUserResponse maps a domain.User to a public-safe UserResponse.
// currencyCode is passed in to avoid an extra DB call when already resolved.
func toUserResponse(user *domain.User, currencyCode string) *UserResponse {
	return &UserResponse{
		ID:                user.ID,
		Email:             user.Email,
		FullName:          user.FullName,
		PreferredCurrency: currencyCode,
		PreferredTimezone: user.PreferredTimezone,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
}

// --- Validators ---

// validateRegisterRequest enforces all registration field rules from the API spec.
func validateRegisterRequest(req RegisterRequest) error {
	email := strings.TrimSpace(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("valid email address is required: %w", domain.ErrInvalidInput)
	}

	if err := validatePassword(req.Password); err != nil {
		return err
	}

	fullName := strings.TrimSpace(req.FullName)
	if len(fullName) < 2 || len(fullName) > 255 {
		return fmt.Errorf("full name must be between 2 and 255 characters: %w", domain.ErrInvalidInput)
	}

	if req.PreferredCurrency == "" || len(req.PreferredCurrency) != 3 {
		return fmt.Errorf("preferred_currency must be a valid 3-letter ISO 4217 code: %w", domain.ErrInvalidInput)
	}

	if req.PreferredTimezone == "" {
		return fmt.Errorf("preferred_timezone is required: %w", domain.ErrInvalidInput)
	}

	return nil
}

// validatePassword enforces: min 8 chars, 1 uppercase, 1 digit, 1 special character.
// These rules match the spec on page 3 of the API contracts document.
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters: %w", domain.ErrInvalidInput)
	}

	var hasUpper, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter: %w", domain.ErrInvalidInput)
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit: %w", domain.ErrInvalidInput)
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character: %w", domain.ErrInvalidInput)
	}

	return nil
}

// validateTimezone verifies the string is a valid IANA timezone.
// Uses time.LoadLocation which rejects invalid strings.
func validateTimezone(tz string) error {
	if _, err := time.LoadLocation(tz); err != nil {
		return fmt.Errorf("preferred_timezone %q is not a valid IANA timezone: %w", tz, domain.ErrInvalidInput)
	}
	return nil
}

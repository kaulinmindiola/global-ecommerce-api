package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a customer in the e-commerce platform.
// It stores user authentication credentials and preferences including
// preferred currency and timezone for personalized experience.
type User struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	FullName          string    `json:"full_name"`
	PasswordHash      string    `json:"-"` // Never serialize password
	PreferredCurrency string    `json:"preferred_currency"`
	PreferredTimezone string    `json:"preferred_timezone"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Validation errors
var (
	ErrInvalidEmail         = errors.New("invalid email format")
	ErrEmailRequired        = errors.New("email is required")
	ErrFullNameRequired     = errors.New("full name is required")
	ErrFullNameTooShort     = errors.New("full name must be at least 2 characters")
	ErrPasswordHashRequired = errors.New("password hash is required")
	ErrInvalidCurrencyCode  = errors.New("invalid currency code (must be 3 uppercase letters)")
	ErrInvalidTimezone      = errors.New("invalid timezone format")
)

var (
	// emailRegex validates email format according to basic RFC 5322
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	// currencyRegex validates ISO 4217 currency codes (3 uppercase letters)
	currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)
)

// NewUser creates a new User with validated fields.
// Returns error if any validation fails.
func NewUser(email, fullName, passwordHash, currency, timezone string) (*User, error) {
	user := &User{
		ID:                uuid.New().String(),
		Email:             strings.TrimSpace(email),
		FullName:          strings.TrimSpace(fullName),
		PasswordHash:      passwordHash,
		PreferredCurrency: strings.ToUpper(strings.TrimSpace(currency)),
		PreferredTimezone: strings.TrimSpace(timezone),
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate checks if the User fields meet business rules.
// Returns the first validation error encountered.
func (u *User) Validate() error {
	// Email validation
	if u.Email == "" {
		return ErrEmailRequired
	}
	if !emailRegex.MatchString(u.Email) {
		return ErrInvalidEmail
	}

	// Full name validation
	if u.FullName == "" {
		return ErrFullNameRequired
	}
	if len(u.FullName) < 2 {
		return ErrFullNameTooShort
	}

	// Password hash validation
	if u.PasswordHash == "" {
		return ErrPasswordHashRequired
	}

	// Currency validation
	if u.PreferredCurrency != "" && !currencyRegex.MatchString(u.PreferredCurrency) {
		return ErrInvalidCurrencyCode
	}

	// Timezone validation (basic check)
	if u.PreferredTimezone == "" {
		u.PreferredTimezone = "UTC" // Default timezone
	}

	return nil
}

// UpdateProfile updates user profile information.
// This method maintains the UpdatedAt timestamp.
func (u *User) UpdateProfile(fullName, currency, timezone string) error {
	if fullName != "" {
		u.FullName = strings.TrimSpace(fullName)
	}
	if currency != "" {
		u.PreferredCurrency = strings.ToUpper(strings.TrimSpace(currency))
	}
	if timezone != "" {
		u.PreferredTimezone = strings.TrimSpace(timezone)
	}

	u.UpdatedAt = time.Now().UTC()

	return u.Validate()
}

// Deactivate marks the user as inactive.
// Inactive users cannot login but data is preserved for audit.
func (u *User) Deactivate() {
	u.IsActive = false
	u.UpdatedAt = time.Now().UTC()
}

// Activate marks the user as active.
func (u *User) Activate() {
	u.IsActive = true
	u.UpdatedAt = time.Now().UTC()
}

package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a customer in the e-commerce platform.
// It stores authentication credentials and user preferences.
//
// Enterprise design note:
// We use PreferredCurrencyID as FK -> currencies.id instead of storing
// only the ISO code string directly. This guarantees referential integrity
// and keeps the domain aligned with PostgreSQL relations.
//
// PreferredCurrencyCode is optional and used mainly for read operations
// (JOIN with currencies table), similar to Product.BaseCurrencyCode.
type User struct {
	ID                    uuid.UUID `json:"id"`
	Email                 string    `json:"email"`
	FullName              string    `json:"full_name"`
	PasswordHash          string    `json:"-"` // Never serialize password
	PreferredCurrencyID   uuid.UUID `json:"preferred_currency_id"`
	PreferredCurrencyCode string    `json:"preferred_currency_code,omitempty"`
	PreferredTimezone     string    `json:"preferred_timezone"`
	IsActive              bool      `json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
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
//
// preferredCurrencyID must reference an existing currency record
// in the currencies table.
func NewUser(
	email,
	fullName,
	passwordHash string,
	preferredCurrencyID uuid.UUID,
	timezone string,
) (*User, error) {
	user := &User{
		ID:                  uuid.New(),
		Email:               strings.TrimSpace(email),
		FullName:            strings.TrimSpace(fullName),
		PasswordHash:        strings.TrimSpace(passwordHash),
		PreferredCurrencyID: preferredCurrencyID,
		PreferredTimezone:   strings.TrimSpace(timezone),
		IsActive:            true,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
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

	// Preferred currency FK validation
	if u.PreferredCurrencyID == uuid.Nil {
		return ErrInvalidCurrencyCode
	}

	// Timezone validation (basic defaulting)
	if strings.TrimSpace(u.PreferredTimezone) == "" {
		u.PreferredTimezone = "UTC"
	}

	return nil
}

// UpdateProfile updates mutable profile fields.
// Currency is updated using FK UUID instead of ISO string.
func (u *User) UpdateProfile(
	fullName string,
	preferredCurrencyID uuid.UUID,
	timezone string,
) error {
	if strings.TrimSpace(fullName) != "" {
		u.FullName = strings.TrimSpace(fullName)
	}

	if preferredCurrencyID != uuid.Nil {
		u.PreferredCurrencyID = preferredCurrencyID
	}

	if strings.TrimSpace(timezone) != "" {
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

// Package validator provides request validation utilities that collect
// all field errors before returning, rather than failing on the first error.
//
// This gives clients a complete picture of what is wrong with their request
// in a single round-trip — a standard expectation in production APIs.
//
// Usage:
//
//	v := validator.New()
//	v.Required("email", req.Email)
//	v.Email("email", req.Email)
//	v.MinLength("password", req.Password, 8)
//	v.ISO4217("currency", req.Currency)
//
//	if v.HasErrors() {
//	    response.Error(w, r, v.ToAppError())
//	    return
//	}
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	apperrors "github.com/kaulinmindiola/global-ecommerce-api/pkg/errors"
)

// Validator collects field-level validation errors across multiple checks.
// It is not safe for concurrent use — create a new instance per request.
type Validator struct {
	errors []apperrors.FieldError
}

// New creates a new empty Validator.
func New() *Validator {
	return &Validator{}
}

// HasErrors returns true if any validation rule has failed.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// ToAppError converts all collected field errors into a single AppError
// suitable for writing as an HTTP response.
func (v *Validator) ToAppError() *apperrors.AppError {
	return apperrors.NewValidationError("Request validation failed", v.errors...)
}

// addError records a field-level validation failure.
func (v *Validator) addError(field, message, code string) {
	v.errors = append(v.errors, apperrors.FieldError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// ── String validators ────────────────────────────────────────────────────────

// Required checks that a string field is non-empty after trimming whitespace.
func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.addError(field, fmt.Sprintf("%s is required", field), "REQUIRED")
	}
	return v
}

// MinLength checks that a string meets the minimum character length.
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.addError(field,
			fmt.Sprintf("%s must be at least %d characters", field, min),
			"TOO_SHORT",
		)
	}
	return v
}

// MaxLength checks that a string does not exceed the maximum character length.
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.addError(field,
			fmt.Sprintf("%s must not exceed %d characters", field, max),
			"TOO_LONG",
		)
	}
	return v
}

// Email validates that a string is a well-formed email address (RFC 5322 simplified).
func (v *Validator) Email(field, value string) *Validator {
	if !emailRegex.MatchString(strings.TrimSpace(value)) {
		v.addError(field, "Must be a valid email address", "INVALID_FORMAT")
	}
	return v
}

// URL validates that a string is a well-formed HTTP/HTTPS URL.
func (v *Validator) URL(field, value string) *Validator {
	if !urlRegex.MatchString(strings.TrimSpace(value)) {
		v.addError(field, "Must be a valid URL (http:// or https://)", "INVALID_FORMAT")
	}
	return v
}

// ── Password validators ──────────────────────────────────────────────────────

// Password enforces the password policy defined in the API spec:
// minimum 8 characters, at least 1 uppercase, 1 digit, 1 special character.
func (v *Validator) Password(field, value string) *Validator {
	if len(value) < 8 {
		v.addError(field, "Password must be at least 8 characters", "TOO_SHORT")
		return v
	}

	var hasUpper, hasDigit, hasSpecial bool
	for _, r := range value {
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
		v.addError(field, "Password must contain at least one uppercase letter", "MISSING_UPPERCASE")
	}
	if !hasDigit {
		v.addError(field, "Password must contain at least one digit", "MISSING_DIGIT")
	}
	if !hasSpecial {
		v.addError(field, "Password must contain at least one special character", "MISSING_SPECIAL")
	}

	return v
}

// ── Numeric validators ───────────────────────────────────────────────────────

// Positive checks that a float64 is strictly greater than zero.
func (v *Validator) Positive(field string, value float64) *Validator {
	if value <= 0 {
		v.addError(field, fmt.Sprintf("%s must be greater than zero", field), "MUST_BE_POSITIVE")
	}
	return v
}

// NonNegative checks that an integer is >= 0.
func (v *Validator) NonNegative(field string, value int) *Validator {
	if value < 0 {
		v.addError(field, fmt.Sprintf("%s cannot be negative", field), "CANNOT_BE_NEGATIVE")
	}
	return v
}

// Min checks that an integer is >= the given minimum.
func (v *Validator) Min(field string, value, min int) *Validator {
	if value < min {
		v.addError(field, fmt.Sprintf("%s must be at least %d", field, min), "TOO_SMALL")
	}
	return v
}

// Max checks that an integer is <= the given maximum.
func (v *Validator) Max(field string, value, max int) *Validator {
	if value > max {
		v.addError(field, fmt.Sprintf("%s must be at most %d", field, max), "TOO_LARGE")
	}
	return v
}

// ── Domain-specific validators ───────────────────────────────────────────────

// ISO4217 validates that a string is a valid 3-letter ISO 4217 currency code.
// Examples: USD, EUR, COP, GBP, JPY
func (v *Validator) ISO4217(field, value string) *Validator {
	code := strings.ToUpper(strings.TrimSpace(value))
	if len(code) != 3 || !isAlpha(code) {
		v.addError(field,
			"Must be a valid 3-letter ISO 4217 currency code (e.g. USD, EUR)",
			"INVALID_CURRENCY_CODE",
		)
	}
	return v
}

// IANATimezone validates that a string is a valid IANA timezone identifier.
// Uses time.LoadLocation which is authoritative for IANA validation.
func (v *Validator) IANATimezone(field, value string) *Validator {
	value = strings.TrimSpace(value)

	if value == "" {
		v.addError(field, "Timezone cannot be empty", "TIMEZONE_EMPTY")
		return v
	}

	if _, err := time.LoadLocation(value); err != nil {
		v.addError(field,
			"Must be a valid IANA timezone (e.g. America/Bogota, Europe/Madrid)",
			"INVALID_TIMEZONE",
		)
	}

	return v
}

// OneOf checks that a string value is one of a set of allowed values.
// The comparison is case-sensitive.
func (v *Validator) OneOf(field, value string, allowed ...string) *Validator {
	for _, a := range allowed {
		if value == a {
			return v
		}
	}
	v.addError(field,
		fmt.Sprintf("Must be one of: %s", strings.Join(allowed, ", ")),
		"INVALID_VALUE",
	)
	return v
}

// UUID validates that a string is a properly formatted UUID v4.
func (v *Validator) UUID(field, value string) *Validator {
	if !uuidRegex.MatchString(value) {
		v.addError(field, "Must be a valid UUID", "INVALID_UUID")
	}
	return v
}

// ── Compiled regex patterns ──────────────────────────────────────────────────
// Compiled once at package init — not per request.

var (
	// emailRegex is a pragmatic RFC 5322 subset — covers 99.9% of real addresses.
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	// urlRegex matches http:// and https:// URLs.
	urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

	// uuidRegex matches a standard UUID v4 format.
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// isAlpha returns true if every rune in s is an ASCII letter.
func isAlpha(s string) bool {
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

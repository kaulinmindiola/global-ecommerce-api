package unit_test

import (
	"testing"

	"github.com/kaulinmindiola/global-ecommerce-api/pkg/validator"
)

func TestValidator_Required(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"non-empty value passes", "hello", false},
		{"empty string fails", "", true},
		{"whitespace-only fails", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			v.Required("field", tt.value)
			if tt.wantErr && !v.HasErrors() {
				t.Error("expected validation error but got none")
			}
			if !tt.wantErr && v.HasErrors() {
				t.Errorf("unexpected validation errors: %v", v.ToAppError().Details)
			}
		})
	}
}

func TestValidator_Email(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"kaulin@example.com", false},
		{"user.name+tag@domain.co.uk", false},
		{"notanemail", true},
		{"@nodomain.com", true},
		{"missing@", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			v := validator.New().Email("email", tt.email)
			if tt.wantErr && !v.HasErrors() {
				t.Errorf("expected email validation to fail for %q", tt.email)
			}
			if !tt.wantErr && v.HasErrors() {
				t.Errorf("expected email validation to pass for %q", tt.email)
			}
		})
	}
}

func TestValidator_Password(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password passes", "SecurePass123!", false},
		{"too short fails", "Short1!", true},
		{"no uppercase fails", "lowercase123!", true},
		{"no digit fails", "NoDigits!!!!", true},
		{"no special char fails", "NoSpecial123", true},
		{"all requirements met", "MyP@ssw0rd", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New().Password("password", tt.password)
			if tt.wantErr && !v.HasErrors() {
				t.Errorf("expected password %q to fail validation", tt.password)
			}
			if !tt.wantErr && v.HasErrors() {
				t.Errorf("expected password %q to pass validation, got: %v",
					tt.password, v.ToAppError().Details)
			}
		})
	}
}

func TestValidator_ISO4217(t *testing.T) {
	tests := []struct {
		code    string
		wantErr bool
	}{
		{"USD", false},
		{"EUR", false},
		{"COP", false},
		{"GBP", false},
		{"usd", false}, // lowercase is normalised
		{"US", true},   // too short
		{"USDD", true}, // too long
		{"123", true},  // digits only
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			v := validator.New().ISO4217("currency", tt.code)
			if tt.wantErr && !v.HasErrors() {
				t.Errorf("expected ISO4217 validation to fail for %q", tt.code)
			}
			if !tt.wantErr && v.HasErrors() {
				t.Errorf("expected ISO4217 validation to pass for %q", tt.code)
			}
		})
	}
}

func TestValidator_IANATimezone(t *testing.T) {
	tests := []struct {
		tz      string
		wantErr bool
	}{
		{"America/Bogota", false},
		{"Europe/Madrid", false},
		{"UTC", false},
		{"America/New_York", false},
		{"Invalid/Timezone", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.tz, func(t *testing.T) {
			v := validator.New().IANATimezone("timezone", tt.tz)
			if tt.wantErr && !v.HasErrors() {
				t.Errorf("expected timezone %q to fail validation", tt.tz)
			}
			if !tt.wantErr && v.HasErrors() {
				t.Errorf("expected timezone %q to pass validation", tt.tz)
			}
		})
	}
}

func TestValidator_CollectsAllErrors(t *testing.T) {
	// Verify that validation does NOT stop on the first error —
	// all field errors must be collected in a single pass.
	v := validator.New()
	v.Required("email", "")
	v.Required("password", "")
	v.ISO4217("currency", "")

	if !v.HasErrors() {
		t.Fatal("expected validation errors")
	}

	appErr := v.ToAppError()
	if len(appErr.Details) != 3 {
		t.Errorf("expected 3 field errors, got %d: %v", len(appErr.Details), appErr.Details)
	}
}

func TestValidator_Positive(t *testing.T) {
	tests := []struct {
		value   float64
		wantErr bool
	}{
		{99.99, false},
		{0.01, false},
		{0, true},
		{-1.0, true},
	}
	for _, tt := range tests {
		v := validator.New().Positive("price", tt.value)
		if tt.wantErr && !v.HasErrors() {
			t.Errorf("expected Positive to fail for %v", tt.value)
		}
		if !tt.wantErr && v.HasErrors() {
			t.Errorf("expected Positive to pass for %v", tt.value)
		}
	}
}

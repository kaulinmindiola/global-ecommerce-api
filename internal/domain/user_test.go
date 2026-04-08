package domain

import (
	"testing"
)

func TestNewUser_Success(t *testing.T) {
	user, err := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		"USD",
		"America/Bogota",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "kaulin@example.com" {
		t.Errorf("expected email 'kaulin@example.com', got '%s'", user.Email)
	}

	if user.PreferredCurrency != "USD" {
		t.Errorf("expected currency 'USD', got '%s'", user.PreferredCurrency)
	}

	if !user.IsActive {
		t.Error("expected user to be active")
	}
}

func TestNewUser_InvalidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"invalid format", "notanemail"},
		{"missing @", "user.example.com"},
		{"missing domain", "user@"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUser(
				tt.email,
				"Kaulin Mindiola",
				"$2a$10$hashedpassword",
				"USD",
				"UTC",
			)

			if err == nil {
				t.Error("expected error for invalid email, got nil")
			}
		})
	}
}

func TestNewUser_InvalidFullName(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		wantErr  error
	}{
		{"empty name", "", ErrFullNameRequired},
		{"too short", "A", ErrFullNameTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewUser(
				"kaulin@example.com",
				tt.fullName,
				"$2a$10$hashedpassword",
				"USD",
				"UTC",
			)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewUser_InvalidCurrency(t *testing.T) {
	_, err := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		"INVALID",
		"UTC",
	)

	if err != ErrInvalidCurrencyCode {
		t.Errorf("expected %v, got %v", ErrInvalidCurrencyCode, err)
	}
}

func TestUser_UpdateProfile(t *testing.T) {
	user, _ := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		"USD",
		"UTC",
	)

	err := user.UpdateProfile("Kaulin A. Mindiola", "EUR", "Europe/Madrid")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.FullName != "Kaulin A. Mindiola" {
		t.Errorf("expected updated name, got '%s'", user.FullName)
	}

	if user.PreferredCurrency != "EUR" {
		t.Errorf("expected currency 'EUR', got '%s'", user.PreferredCurrency)
	}
}

func TestUser_Deactivate(t *testing.T) {
	user, _ := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		"USD",
		"UTC",
	)

	user.Deactivate()

	if user.IsActive {
		t.Error("expected user to be inactive after deactivation")
	}
}

func TestUser_Activate(t *testing.T) {
	user, _ := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		"USD",
		"UTC",
	)

	user.Deactivate()
	user.Activate()

	if !user.IsActive {
		t.Error("expected user to be active after activation")
	}
}

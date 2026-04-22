package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUser_Success(t *testing.T) {
	currencyID := uuid.New()

	user, err := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		currencyID,
		"America/Bogota",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.Email != "kaulin@example.com" {
		t.Errorf("expected email 'kaulin@example.com', got '%s'", user.Email)
	}

	if user.PreferredCurrencyID != currencyID {
		t.Errorf("expected preferred currency ID %v, got %v",
			currencyID,
			user.PreferredCurrencyID,
		)
	}

	if !user.IsActive {
		t.Error("expected user to be active")
	}
}

func TestNewUser_InvalidEmail(t *testing.T) {
	currencyID := uuid.New()

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
				currencyID,
				"UTC",
			)

			if err == nil {
				t.Error("expected error for invalid email, got nil")
			}
		})
	}
}

func TestNewUser_InvalidFullName(t *testing.T) {
	currencyID := uuid.New()

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
				currencyID,
				"UTC",
			)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewUser_InvalidCurrency(t *testing.T) {
	// uuid.Nil simula ausencia de FK válida
	_, err := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		uuid.Nil,
		"UTC",
	)

	if err != ErrInvalidCurrencyCode {
		t.Errorf("expected %v, got %v", ErrInvalidCurrencyCode, err)
	}
}

func TestUser_UpdateProfile(t *testing.T) {
	initialCurrencyID := uuid.New()
	newCurrencyID := uuid.New()

	user, _ := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		initialCurrencyID,
		"UTC",
	)

	err := user.UpdateProfile(
		"Kaulin A. Mindiola",
		newCurrencyID,
		"Europe/Madrid",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user.FullName != "Kaulin A. Mindiola" {
		t.Errorf("expected updated name, got '%s'", user.FullName)
	}

	if user.PreferredCurrencyID != newCurrencyID {
		t.Errorf("expected currency ID %v, got %v",
			newCurrencyID,
			user.PreferredCurrencyID,
		)
	}
}

func TestUser_Deactivate(t *testing.T) {
	currencyID := uuid.New()

	user, _ := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		currencyID,
		"UTC",
	)

	user.Deactivate()

	if user.IsActive {
		t.Error("expected user to be inactive after deactivation")
	}
}

func TestUser_Activate(t *testing.T) {
	currencyID := uuid.New()

	user, _ := NewUser(
		"kaulin@example.com",
		"Kaulin Mindiola",
		"$2a$10$hashedpassword",
		currencyID,
		"UTC",
	)

	user.Deactivate()
	user.Activate()

	if !user.IsActive {
		t.Error("expected user to be active after activation")
	}
}

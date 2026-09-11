package domain

import (
	"testing"
)

func TestNewCurrency_Success(t *testing.T) {
	currency, err := NewCurrency("USD", "US Dollar", "$", 2)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if currency.Code != "USD" {
		t.Errorf("expected code 'USD', got '%s'", currency.Code)
	}

	if currency.Name != "US Dollar" {
		t.Errorf("expected name 'US Dollar', got '%s'", currency.Name)
	}

	if currency.Symbol != "$" {
		t.Errorf("expected symbol '$', got '%s'", currency.Symbol)
	}

	if currency.DecimalPlaces != 2 {
		t.Errorf("expected decimal places 2, got %d", currency.DecimalPlaces)
	}

	if !currency.IsActive {
		t.Error("expected currency to be active")
	}
}

func TestNewCurrency_InvalidCode(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"empty code", ""},
		{"too short", "US"},
		{"too long", "USDD"},
		{"lowercase", "usd"},
		{"numbers", "U5D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCurrency(tt.code, "Currency", "$", 2)

			if err == nil {
				t.Error("expected error for invalid code, got nil")
			}
		})
	}
}

func TestNewCurrency_InvalidDecimalPlaces(t *testing.T) {
	tests := []struct {
		name          string
		decimalPlaces int
	}{
		{"negative", -1},
		{"too many", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCurrency("USD", "US Dollar", "$", tt.decimalPlaces)

			if err != ErrCurrencyDecimalInvalid {
				t.Errorf("expected %v, got %v", ErrCurrencyDecimalInvalid, err)
			}
		})
	}
}

func TestNewExchangeRate_Success(t *testing.T) {
	rate, err := NewExchangeRate("USD", "EUR", 0.92, "exchangerate-api")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if rate.FromCurrency != "USD" {
		t.Errorf("expected from currency 'USD', got '%s'", rate.FromCurrency)
	}

	if rate.ToCurrency != "EUR" {
		t.Errorf("expected to currency 'EUR', got '%s'", rate.ToCurrency)
	}

	if rate.Rate != 0.92 {
		t.Errorf("expected rate 0.92, got %f", rate.Rate)
	}
}

func TestNewExchangeRate_SameCurrency(t *testing.T) {
	_, err := NewExchangeRate("USD", "USD", 1.0, "test")

	if err != ErrExchangeRateSameCurrency {
		t.Errorf("expected %v, got %v", ErrExchangeRateSameCurrency, err)
	}
}

func TestNewExchangeRate_InvalidRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"zero", 0.0},
		{"negative", -0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewExchangeRate("USD", "EUR", tt.rate, "test")

			if err != ErrExchangeRateInvalid {
				t.Errorf("expected %v, got %v", ErrExchangeRateInvalid, err)
			}
		})
	}
}

func TestExchangeRate_Convert(t *testing.T) {
	rate, _ := NewExchangeRate("USD", "EUR", 0.92, "test")

	tests := []struct {
		name   string
		amount float64
		want   float64
	}{
		{"simple conversion", 100.00, 92.00},
		{"decimal amount", 50.50, 46.46},
		{"large amount", 1000.00, 920.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rate.Convert(tt.amount)

			if got != tt.want {
				t.Errorf("expected %f, got %f", tt.want, got)
			}
		})
	}
}

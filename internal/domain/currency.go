package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Currency represents a supported currency in the system.
// Follows ISO 4217 standard for currency codes.
type Currency struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`           // ISO 4217 code (USD, EUR, etc.)
	Name          string    `json:"name"`           // Full name (US Dollar, Euro, etc.)
	Symbol        string    `json:"symbol"`         // Currency symbol ($, €, etc.)
	DecimalPlaces int       `json:"decimal_places"` // Number of decimal places (2 for most, 0 for JPY)
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Currency validation errors
var (
	ErrCurrencyCodeRequired   = errors.New("currency code is required")
	ErrCurrencyCodeInvalid    = errors.New("currency code must be 3 uppercase letters")
	ErrCurrencyNameRequired   = errors.New("currency name is required")
	ErrCurrencySymbolRequired = errors.New("currency symbol is required")
	ErrCurrencyDecimalInvalid = errors.New("decimal places must be between 0 and 4")
)

// NewCurrency creates a new Currency with validated fields.
func NewCurrency(code, name, symbol string, decimalPlaces int) (*Currency, error) {
	currency := &Currency{
		ID:            uuid.New().String(),
		Code:          strings.TrimSpace(code),
		Name:          strings.TrimSpace(name),
		Symbol:        strings.TrimSpace(symbol),
		DecimalPlaces: decimalPlaces,
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if err := currency.Validate(); err != nil {
		return nil, err
	}

	return currency, nil
}

// Validate checks if the Currency fields meet business rules.
func (c *Currency) Validate() error {
	// Code validation
	if c.Code == "" {
		return ErrCurrencyCodeRequired
	}
	if !currencyRegex.MatchString(c.Code) {
		return ErrCurrencyCodeInvalid
	}

	// Name validation
	if c.Name == "" {
		return ErrCurrencyNameRequired
	}

	// Symbol validation
	if c.Symbol == "" {
		return ErrCurrencySymbolRequired
	}

	// Decimal places validation
	if c.DecimalPlaces < 0 || c.DecimalPlaces > 4 {
		return ErrCurrencyDecimalInvalid
	}

	return nil
}

// Deactivate marks the currency as inactive.
// Inactive currencies cannot be used for new transactions.
func (c *Currency) Deactivate() {
	c.IsActive = false
	c.UpdatedAt = time.Now().UTC()
}

// Activate marks the currency as active.
func (c *Currency) Activate() {
	c.IsActive = true
	c.UpdatedAt = time.Now().UTC()
}

// ExchangeRate represents a conversion rate between two currencies.
type ExchangeRate struct {
	ID            string    `json:"id"`
	FromCurrency  string    `json:"from_currency"`
	ToCurrency    string    `json:"to_currency"`
	Rate          float64   `json:"rate"`
	EffectiveDate time.Time `json:"effective_date"`
	Source        string    `json:"source"` // API source (e.g., "exchangerate-api")
	CreatedAt     time.Time `json:"created_at"`
}

// ExchangeRate validation errors
var (
	ErrExchangeRateInvalid      = errors.New("exchange rate must be greater than zero")
	ErrExchangeRateSameCurrency = errors.New("from and to currency must be different")
)

// NewExchangeRate creates a new ExchangeRate with validated fields.
func NewExchangeRate(fromCurrency, toCurrency string, rate float64, source string) (*ExchangeRate, error) {
	exchangeRate := &ExchangeRate{
		ID:            uuid.New().String(),
		FromCurrency:  strings.ToUpper(strings.TrimSpace(fromCurrency)),
		ToCurrency:    strings.ToUpper(strings.TrimSpace(toCurrency)),
		Rate:          rate,
		EffectiveDate: time.Now().UTC(),
		Source:        source,
		CreatedAt:     time.Now().UTC(),
	}

	if err := exchangeRate.Validate(); err != nil {
		return nil, err
	}

	return exchangeRate, nil
}

// Validate checks if the ExchangeRate fields meet business rules.
func (e *ExchangeRate) Validate() error {
	// Rate validation
	if e.Rate <= 0 {
		return ErrExchangeRateInvalid
	}

	// Currency codes validation
	if e.FromCurrency == e.ToCurrency {
		return ErrExchangeRateSameCurrency
	}

	if !currencyRegex.MatchString(e.FromCurrency) {
		return ErrCurrencyCodeInvalid
	}

	if !currencyRegex.MatchString(e.ToCurrency) {
		return ErrCurrencyCodeInvalid
	}

	return nil
}

// Convert applies the exchange rate to an amount.
func (e *ExchangeRate) Convert(amount float64) float64 {
	return amount * e.Rate
}

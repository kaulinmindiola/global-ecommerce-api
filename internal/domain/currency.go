package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Currency represents a supported currency in the system.
// Follows ISO 4217 standard for currency codes and supports
// financial exchange-rate operations for global e-commerce.
type Currency struct {
	ID                uuid.UUID `json:"id"`
	Code              string    `json:"code"`                 // ISO 4217 code (USD, EUR, COP)
	Name              string    `json:"name"`                 // Full currency name
	Symbol            string    `json:"symbol"`               // Currency symbol ($, €, £)
	DecimalPlaces     int       `json:"decimal_places"`       // Usually 2, JPY = 0
	ExchangeRateToUSD float64   `json:"exchange_rate_to_usd"` // Relative conversion rate to USD
	IsActive          bool      `json:"is_active"`            // Soft enable/disable
	RateUpdatedAt     time.Time `json:"rate_updated_at"`      // Last FX update timestamp
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

//
// Domain Errors
//

var (
	// Generic domain errors

	// Currency validation errors
	ErrCurrencyCodeRequired   = errors.New("currency code is required")
	ErrCurrencyCodeInvalid    = errors.New("currency code must be 3 uppercase letters")
	ErrCurrencyNameRequired   = errors.New("currency name is required")
	ErrCurrencySymbolRequired = errors.New("currency symbol is required")
	ErrCurrencyDecimalInvalid = errors.New("decimal places must be between 0 and 4")

	// Exchange rate validation errors
	ErrExchangeRateInvalid      = errors.New("exchange rate must be greater than zero")
	ErrExchangeRateSameCurrency = errors.New("from and to currency must be different")
)

//
// Currency Constructor
//

// NewCurrency creates a new Currency with validated business rules.
//
// Default rules:
// - Code is normalized to uppercase
// - ExchangeRateToUSD starts at 1.0 by default
// - Currency starts active
// - RateUpdatedAt initialized immediately
func NewCurrency(
	code string,
	name string,
	symbol string,
	decimalPlaces int,
) (*Currency, error) {
	now := time.Now().UTC()

	currency := &Currency{
		ID:                uuid.New(),
		Code:              strings.ToUpper(strings.TrimSpace(code)),
		Name:              strings.TrimSpace(name),
		Symbol:            strings.TrimSpace(symbol),
		DecimalPlaces:     decimalPlaces,
		ExchangeRateToUSD: 1.0,
		IsActive:          true,
		RateUpdatedAt:     now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := currency.Validate(); err != nil {
		return nil, err
	}

	return currency, nil
}

//
// Currency Validation
//

// Validate checks if Currency satisfies business constraints.
func (c *Currency) Validate() error {
	// ISO code validation
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

//
// Domain Behaviors
//

// Deactivate disables the currency for future transactions.
func (c *Currency) Deactivate() {
	c.IsActive = false
	c.UpdatedAt = time.Now().UTC()
}

// Activate enables the currency for transactions.
func (c *Currency) Activate() {
	c.IsActive = true
	c.UpdatedAt = time.Now().UTC()
}

// UpdateExchangeRate updates the conversion rate relative to USD.
//
// This is a domain behavior and should be used by services
// instead of directly mutating the field.
func (c *Currency) UpdateExchangeRate(rate float64) error {
	if rate <= 0 {
		return ErrExchangeRateInvalid
	}

	now := time.Now().UTC()

	c.ExchangeRateToUSD = rate
	c.RateUpdatedAt = now
	c.UpdatedAt = now

	return nil
}

//
// ExchangeRate Entity
//

// ExchangeRate represents a conversion rate between two currencies.
//
// This is useful when storing external FX history
// independently from the base Currency entity.
type ExchangeRate struct {
	ID            uuid.UUID `json:"id"`
	FromCurrency  string    `json:"from_currency"`
	ToCurrency    string    `json:"to_currency"`
	Rate          float64   `json:"rate"`
	EffectiveDate time.Time `json:"effective_date"`
	Source        string    `json:"source"` // Example: exchangerate-api
	CreatedAt     time.Time `json:"created_at"`
}

//
// ExchangeRate Constructor
//

// NewExchangeRate creates a validated exchange rate entity.
func NewExchangeRate(
	fromCurrency string,
	toCurrency string,
	rate float64,
	source string,
) (*ExchangeRate, error) {
	now := time.Now().UTC()

	exchangeRate := &ExchangeRate{
		ID:            uuid.New(),
		FromCurrency:  strings.ToUpper(strings.TrimSpace(fromCurrency)),
		ToCurrency:    strings.ToUpper(strings.TrimSpace(toCurrency)),
		Rate:          rate,
		EffectiveDate: now,
		Source:        strings.TrimSpace(source),
		CreatedAt:     now,
	}

	if err := exchangeRate.Validate(); err != nil {
		return nil, err
	}

	return exchangeRate, nil
}

//
// ExchangeRate Validation
//

// Validate ensures ExchangeRate satisfies business constraints.
func (e *ExchangeRate) Validate() error {
	if e.Rate <= 0 {
		return ErrExchangeRateInvalid
	}

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

//
// ExchangeRate Behavior
//

// Convert applies the exchange rate to a monetary amount.
func (e *ExchangeRate) Convert(amount float64) float64 {
	return amount * e.Rate
}

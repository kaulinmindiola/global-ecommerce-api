package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
)

// CurrencyService defines the business operations for currency management
// and monetary conversion. All conversion math routes through this service
// to guarantee a single, auditable implementation of exchange rate logic.
type CurrencyService interface {
	// ListActive returns all currently supported currencies.
	ListActive(ctx context.Context) ([]*domain.Currency, error)

	// GetByCode retrieves a single currency by its ISO 4217 code.
	GetByCode(ctx context.Context, code string) (*domain.Currency, error)

	// Convert performs a monetary conversion between two currencies.
	// All conversions pivot through USD as the base unit:
	//   amount (fromCode) → USD → amount (toCode)
	// This guarantees mathematical consistency across all currency pairs.
	Convert(ctx context.Context, req ConvertRequest) (*ConvertResult, error)

	// ConvertAmount converts a raw decimal amount using a known exchange rate.
	// Used internally by OrderService to price order items without extra DB calls.
	ConvertAmount(amount float64, exchangeRate float64) float64

	// GetExchangeRate resolves the effective exchange rate from one currency to another.
	// Rate is always expressed as: 1 unit of fromCode = N units of toCode.
	GetExchangeRate(ctx context.Context, fromCode, toCode string) (float64, error)
}

// ConvertRequest is the input DTO for a currency conversion operation.
type ConvertRequest struct {
	FromCode string
	ToCode   string
	Amount   float64
}

// ConvertResult is the output DTO returned to the handler layer.
// It carries all metadata a client needs to display and audit a conversion.
type ConvertResult struct {
	FromCurrency    string    `json:"from_currency"`
	ToCurrency      string    `json:"to_currency"`
	Amount          float64   `json:"amount"`
	ConvertedAmount float64   `json:"converted_amount"`
	ExchangeRate    float64   `json:"exchange_rate"`
	RateDate        string    `json:"rate_date"`
	Timestamp       time.Time `json:"timestamp"`
}

// currencyService is the concrete implementation of CurrencyService.
type currencyService struct {
	currencyRepo repository.CurrencyRepository
	cache        repository.CacheRepository

	// cacheTTL controls how long active currency lists are cached.
	// Exchange rates change infrequently; 5 minutes is a safe default.
	cacheTTL time.Duration
}

// NewCurrencyService constructs a CurrencyService with its required dependencies.
func NewCurrencyService(
	currencyRepo repository.CurrencyRepository,
	cache repository.CacheRepository,
) CurrencyService {
	return &currencyService{
		currencyRepo: currencyRepo,
		cache:        cache,
		cacheTTL:     5 * time.Minute,
	}
}

// ListActive returns all active currencies, serving from cache when available.
// Cache key: "currencies:active"
func (s *currencyService) ListActive(ctx context.Context) ([]*domain.Currency, error) {
	const cacheKey = "currencies:active"

	// Attempt cache hit first.
	var cached []*domain.Currency
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return cached, nil
	}

	// Cache miss — fetch from database.
	currencies, err := s.currencyRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing active currencies: %w", err)
	}

	// Populate cache asynchronously — a failure here must not block the response.
	_ = s.cache.Set(ctx, cacheKey, currencies, s.cacheTTL)

	return currencies, nil
}

// GetByCode retrieves a currency by ISO code with a per-currency cache entry.
// Cache key pattern: "currency:code:{CODE}"
func (s *currencyService) GetByCode(ctx context.Context, code string) (*domain.Currency, error) {
	cacheKey := fmt.Sprintf("currency:code:%s", code)

	var cached domain.Currency
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	currency, err := s.currencyRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("getting currency %q: %w", code, err)
	}

	_ = s.cache.Set(ctx, cacheKey, currency, s.cacheTTL)

	return currency, nil
}

// Convert performs a full monetary conversion and returns a structured result.
//
// Conversion algorithm (USD as pivot):
//
//	convertedAmount = amount * (toCurrency.ExchangeRateToUSD / fromCurrency.ExchangeRateToUSD)
//
// Special case: if fromCode == "USD", the rate is simply toCurrency.ExchangeRateToUSD.
// If toCode == "USD", the rate is 1 / fromCurrency.ExchangeRateToUSD.
//
// All results are rounded to 4 decimal places for display precision without
// introducing floating-point drift in subsequent calculations.
func (s *currencyService) Convert(ctx context.Context, req ConvertRequest) (*ConvertResult, error) {
	if req.Amount <= 0 {
		return nil, domain.ErrInvalidInput
	}

	rate, err := s.GetExchangeRate(ctx, req.FromCode, req.ToCode)
	if err != nil {
		return nil, err
	}

	convertedAmount := s.ConvertAmount(req.Amount, rate)

	// Fetch rate date from the target currency for audit transparency.
	toCurrency, err := s.GetByCode(ctx, req.ToCode)
	if err != nil {
		return nil, err
	}

	return &ConvertResult{
		FromCurrency:    req.FromCode,
		ToCurrency:      req.ToCode,
		Amount:          roundToDecimalPlaces(req.Amount, 4),
		ConvertedAmount: convertedAmount,
		ExchangeRate:    rate,
		RateDate:        toCurrency.RateUpdatedAt.Format("2006-01-02"),
		Timestamp:       time.Now().UTC(),
	}, nil
}

// ConvertAmount applies an exchange rate to an amount and rounds to 4 decimal places.
// This is the single source of truth for all monetary math in the system.
func (s *currencyService) ConvertAmount(amount float64, exchangeRate float64) float64 {
	return roundToDecimalPlaces(amount*exchangeRate, 4)
}

// GetExchangeRate resolves the effective rate between two currency codes.
// Rate semantics: 1 unit of fromCode = N units of toCode.
func (s *currencyService) GetExchangeRate(ctx context.Context, fromCode, toCode string) (float64, error) {
	// Trivial case — same currency, no conversion needed.
	if fromCode == toCode {
		return 1.0, nil
	}

	fromCurrency, err := s.GetByCode(ctx, fromCode)
	if err != nil {
		return 0, fmt.Errorf("resolving source currency %q: %w", fromCode, err)
	}

	toCurrency, err := s.GetByCode(ctx, toCode)
	if err != nil {
		return 0, fmt.Errorf("resolving target currency %q: %w", toCode, err)
	}

	// Guard against a zero rate in the database — this would cause a division by zero.
	if fromCurrency.ExchangeRateToUSD == 0 {
		return 0, fmt.Errorf("invalid exchange rate for currency %q: rate is zero", fromCode)
	}

	// Cross-rate calculation: pivot through USD.
	rate := toCurrency.ExchangeRateToUSD / fromCurrency.ExchangeRateToUSD

	return roundToDecimalPlaces(rate, 6), nil
}

// roundToDecimalPlaces rounds a float64 to n decimal places.
// Uses math.Round to avoid floating-point representation errors.
func roundToDecimalPlaces(val float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(val*factor) / factor
}

// InvalidateCurrencyCache removes all currency-related cache entries.
// Called after an admin updates exchange rates to ensure fresh data is served.
func InvalidateCurrencyCache(ctx context.Context, cache repository.CacheRepository, currencyID uuid.UUID) {
	_ = cache.DeleteByPattern(ctx, "currencies:*")
	_ = cache.DeleteByPattern(ctx, "currency:code:*")
	_ = cache.Delete(ctx, fmt.Sprintf("currency:id:%s", currencyID))
}

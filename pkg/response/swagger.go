package response

import (
	"time"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
)

// ============================================================================
// HEALTH RESPONSES
// ============================================================================

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string            `json:"status" example:"healthy"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version" example:"1.0.0"`
	Services  map[string]string `json:"services"`
}

// VersionResponse represents version information
type VersionResponse struct {
	Version    string `json:"version" example:"1.0.0"`
	BuildDate  string `json:"build_date"`
	GoVersion  string `json:"go_version" example:"1.22"`
	CommitHash string `json:"commit_hash" example:"a1b2c3d"`
}

// ============================================================================
// AUTHENTICATION RESPONSES
// ============================================================================

// AuthResponse represents authentication response with user and token
type AuthResponse struct {
	User  *domain.User  `json:"user"`
	Token *TokenDetails `json:"token"`
}

// TokenDetails contains JWT token information
type TokenDetails struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int64  `json:"expires_in" example:"86400"`
}

// TokenResponse for token refresh
type TokenResponse struct {
	Token *TokenDetails `json:"token"`
}

// ============================================================================
// CURRENCY RESPONSES
// ============================================================================

// CurrencyListResponse represents list of currencies
type CurrencyListResponse struct {
	Data  []*domain.Currency `json:"data"`
	Total int                `json:"total"`
}

// ConversionResponse represents currency conversion result
type ConversionResponse struct {
	FromCurrency    string    `json:"from_currency" example:"USD"`
	ToCurrency      string    `json:"to_currency" example:"EUR"`
	Amount          float64   `json:"amount" example:"100.00"`
	ConvertedAmount float64   `json:"converted_amount" example:"92.50"`
	ExchangeRate    float64   `json:"exchange_rate" example:"0.925"`
	RateDate        string    `json:"rate_date" example:"2025-01-15"`
	Timestamp       time.Time `json:"timestamp"`
}

// ExchangeRatesResponse represents all exchange rates
type ExchangeRatesResponse struct {
	BaseCurrency string             `json:"base_currency" example:"USD"`
	Rates        map[string]float64 `json:"rates"`
	Timestamp    time.Time          `json:"timestamp"`
}

// ============================================================================
// PRODUCT RESPONSES
// ============================================================================

// ProductListResponse represents paginated product list
type ProductListResponse struct {
	Data       []*domain.Product `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

// ============================================================================
// ORDER RESPONSES
// ============================================================================

// OrderListResponse represents paginated order list
type OrderListResponse struct {
	Data       []*OrderSummary `json:"data"`
	Pagination Pagination      `json:"pagination"`
}

// OrderSummary represents order in list view
type OrderSummary struct {
	ID          string    `json:"id"`
	OrderNumber string    `json:"order_number" example:"ORD-2025-000123"`
	TotalAmount float64   `json:"total_amount" example:"150.00"`
	Currency    string    `json:"currency" example:"USD"`
	Status      string    `json:"status" example:"pending"`
	ItemsCount  int       `json:"items_count" example:"3"`
	CreatedAt   time.Time `json:"created_at"`
}

// ============================================================================
// COMMON RESPONSES
// ============================================================================

// Pagination represents pagination metadata
type Pagination struct {
	Page       int `json:"page" example:"1"`
	Limit      int `json:"limit" example:"20"`
	Total      int `json:"total" example:"100"`
	TotalPages int `json:"total_pages" example:"5"`
}

// ErrorResponse represents API error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code      string      `json:"code" example:"VALIDATION_ERROR"`
	Message   string      `json:"message" example:"Validation failed"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Path      string      `json:"path" example:"/api/v1/orders"`
	RequestID string      `json:"request_id,omitempty" example:"req_abc123xyz"`
}

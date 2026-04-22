package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Product represents a product in the global e-commerce catalog.
//
// Architecture notes:
// - BasePrice uses decimal.Decimal to avoid floating-point errors
// - BaseCurrencyID stores FK reference to currencies table
// - BaseCurrencyCode is populated via JOIN for read operations
// - StockQuantity is persisted inventory quantity
type Product struct {
	ID               uuid.UUID       `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	SKU              string          `json:"sku"`
	BasePrice        decimal.Decimal `json:"base_price"`
	BaseCurrencyID   uuid.UUID       `json:"base_currency_id"`
	BaseCurrencyCode string          `json:"base_currency_code,omitempty"`
	StockQuantity    int             `json:"stock_quantity"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

//
// Domain Errors
//

var (
	ErrProductNameRequired    = errors.New("product name is required")
	ErrProductNameTooShort    = errors.New("product name must be at least 2 characters")
	ErrProductSKURequired     = errors.New("product SKU is required")
	ErrProductSKUInvalid      = errors.New("product SKU must be alphanumeric with hyphens")
	ErrProductPriceInvalid    = errors.New("product price must be greater than zero")
	ErrProductCurrencyInvalid = errors.New("product base currency is invalid")
	ErrProductStockNegative   = errors.New("product stock quantity cannot be negative")
	//ErrInsufficientStock      = errors.New("insufficient stock")
)

//
// Constructor
//

// NewProduct creates a new validated product.
//
// Notes:
// - SKU is normalized to uppercase
// - Product starts active
// - BaseCurrencyCode is optional and usually filled by repository JOINs
func NewProduct(
	name string,
	description string,
	sku string,
	basePrice decimal.Decimal,
	baseCurrencyID uuid.UUID,
	stockQuantity int,
) (*Product, error) {
	now := time.Now().UTC()

	product := &Product{
		ID:             uuid.New(),
		Name:           strings.TrimSpace(name),
		Description:    strings.TrimSpace(description),
		SKU:            strings.ToUpper(strings.TrimSpace(sku)),
		BasePrice:      basePrice,
		BaseCurrencyID: baseCurrencyID,
		StockQuantity:  stockQuantity,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := product.Validate(); err != nil {
		return nil, err
	}

	return product, nil
}

//
// Validation
//

// Validate enforces domain business rules.
func (p *Product) Validate() error {
	// Name validation
	if p.Name == "" {
		return ErrProductNameRequired
	}

	if len(p.Name) < 2 {
		return ErrProductNameTooShort
	}

	// SKU validation
	if p.SKU == "" {
		return ErrProductSKURequired
	}

	if !isValidSKU(p.SKU) {
		return ErrProductSKUInvalid
	}

	// Price validation
	if p.BasePrice.LessThanOrEqual(decimal.Zero) {
		return ErrProductPriceInvalid
	}

	// Currency validation
	if p.BaseCurrencyID == uuid.Nil {
		return ErrProductCurrencyInvalid
	}

	// Stock validation
	if p.StockQuantity < 0 {
		return ErrProductStockNegative
	}

	return nil
}

//
// Domain Behaviors
//

// UpdateStock adjusts stock quantity safely.
func (p *Product) UpdateStock(delta int) error {
	newStock := p.StockQuantity + delta

	if newStock < 0 {
		return ErrProductStockNegative
	}

	p.StockQuantity = newStock
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// ReserveStock reserves stock for an order.
func (p *Product) ReserveStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	if p.StockQuantity < quantity {
		return ErrInsufficientStock
	}

	p.StockQuantity -= quantity
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// RestoreStock restores stock after cancellation/refund.
func (p *Product) RestoreStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	p.StockQuantity += quantity
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// UpdatePrice updates product price.
func (p *Product) UpdatePrice(newPrice decimal.Decimal) error {
	if newPrice.LessThanOrEqual(decimal.Zero) {
		return ErrProductPriceInvalid
	}

	p.BasePrice = newPrice
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// Deactivate performs soft disable.
func (p *Product) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now().UTC()
}

// Activate enables the product.
func (p *Product) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now().UTC()
}

// IsInStock returns true if inventory exists.
func (p *Product) IsInStock() bool {
	return p.StockQuantity > 0
}

// CanFulfillOrder validates enough stock.
func (p *Product) CanFulfillOrder(quantity int) bool {
	return p.StockQuantity >= quantity
}

//
// Helpers
//

// isValidSKU validates uppercase alphanumeric SKU + hyphen.
func isValidSKU(sku string) bool {
	for _, char := range sku {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return false
		}
	}

	return len(sku) > 0
}

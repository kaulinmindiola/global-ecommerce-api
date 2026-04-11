package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Product represents an item in the e-commerce catalog.
// It stores pricing in a base currency and maintains inventory levels.
type Product struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	SKU          string          `json:"sku"`
	BasePrice    decimal.Decimal `json:"base_price"`
	BaseCurrency string          `json:"base_currency"`
	StockQty     int             `json:"stock_quantity"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Product validation errors
var (
	ErrProductNameRequired    = errors.New("product name is required")
	ErrProductNameTooShort    = errors.New("product name must be at least 2 characters")
	ErrProductSKURequired     = errors.New("product SKU is required")
	ErrProductSKUInvalid      = errors.New("product SKU must be alphanumeric with hyphens")
	ErrProductPriceInvalid    = errors.New("product price must be greater than zero")
	ErrProductCurrencyInvalid = errors.New("product base currency must be a valid 3-letter code")
	ErrProductStockNegative   = errors.New("product stock quantity cannot be negative")
	ErrInsufficientStock      = errors.New("insufficient stock for requested quantity")
)

// NewProduct creates a new Product with validated fields.
func NewProduct(name, description, sku string, basePrice decimal.Decimal, baseCurrency string, stockQty int) (*Product, error) {
	product := &Product{
		ID:           uuid.New().String(),
		Name:         strings.TrimSpace(name),
		Description:  strings.TrimSpace(description),
		SKU:          strings.TrimSpace(sku),
		BasePrice:    basePrice,
		BaseCurrency: strings.TrimSpace(baseCurrency),
		StockQty:     stockQty,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := product.Validate(); err != nil {
		return nil, err
	}

	return product, nil
}

// Validate checks if the Product fields meet business rules.
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
	if !currencyRegex.MatchString(p.BaseCurrency) {
		return ErrProductCurrencyInvalid
	}

	// Stock validation
	if p.StockQty < 0 {
		return ErrProductStockNegative
	}

	return nil
}

// UpdateStock adjusts the stock quantity.
// Returns error if the adjustment would result in negative stock.
func (p *Product) UpdateStock(adjustment int) error {
	newStock := p.StockQty + adjustment

	if newStock < 0 {
		return ErrProductStockNegative
	}

	p.StockQty = newStock
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// ReserveStock decreases stock for an order.
// This method should be called within a database transaction.
func (p *Product) ReserveStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	if p.StockQty < quantity {
		return ErrInsufficientStock
	}

	p.StockQty -= quantity
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// RestoreStock increases stock (e.g., when order is cancelled).
func (p *Product) RestoreStock(quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	p.StockQty += quantity
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// UpdatePrice changes the base price of the product.
func (p *Product) UpdatePrice(newPrice decimal.Decimal) error {
	if newPrice.LessThanOrEqual(decimal.Zero) {
		return ErrProductPriceInvalid
	}

	p.BasePrice = newPrice
	p.UpdatedAt = time.Now().UTC()

	return nil
}

// Deactivate marks the product as inactive (e.g., discontinued).
func (p *Product) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now().UTC()
}

// Activate marks the product as active.
func (p *Product) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now().UTC()
}

// IsInStock returns true if the product has available stock.
func (p *Product) IsInStock() bool {
	return p.StockQty > 0
}

// CanFulfillOrder checks if there's enough stock for a given quantity.
func (p *Product) CanFulfillOrder(quantity int) bool {
	return p.StockQty >= quantity
}

// isValidSKU checks if SKU contains only alphanumeric characters and hyphens.
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

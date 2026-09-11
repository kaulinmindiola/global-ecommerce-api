package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderItem represents a single product line in an order.
// It captures product details at the time of purchase for audit trail.
type OrderItem struct {
	ID          uuid.UUID       `json:"id"`
	OrderID     uuid.UUID       `json:"order_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"` // Snapshot at time of purchase
	ProductSKU  string          `json:"product_sku"`  // Snapshot at time of purchase
	UnitPrice   decimal.Decimal `json:"unit_price"`   // Price in order currency
	Quantity    int             `json:"quantity"`
	Subtotal    decimal.Decimal `json:"subtotal"` // UnitPrice * Quantity
	CreatedAt   time.Time       `json:"created_at"`
}

// OrderItem validation errors
var (
	ErrOrderItemProductIDRequired = errors.New("order item product ID is required")
	ErrOrderItemNameRequired      = errors.New("order item product name is required")
	ErrOrderItemSKURequired       = errors.New("order item product SKU is required")
	ErrOrderItemPriceInvalid      = errors.New("order item unit price must be greater than zero")
	ErrOrderItemQuantityInvalid   = errors.New("order item quantity must be at least 1")
)

// NewOrderItem creates a new OrderItem with validated fields.
func NewOrderItem(
	orderID uuid.UUID,
	productID uuid.UUID,
	productName string,
	productSKU string,
	unitPrice decimal.Decimal,
	quantity int,
) (*OrderItem, error) {
	item := &OrderItem{
		ID:          uuid.New(),
		OrderID:     orderID,
		ProductID:   productID,
		ProductName: productName,
		ProductSKU:  productSKU,
		UnitPrice:   unitPrice,
		Quantity:    quantity,
		Subtotal:    decimal.Zero,
		CreatedAt:   time.Now().UTC(),
	}

	// Calculate subtotal before validation
	item.CalculateSubtotal()

	if err := item.Validate(); err != nil {
		return nil, err
	}

	return item, nil
}

// NewOrderItemFromProduct creates an OrderItem from a Product.
// This is a convenience method that automatically captures product details.
func NewOrderItemFromProduct(
	orderID uuid.UUID,
	product *Product,
	quantity int,
	exchangeRate decimal.Decimal,
) (*OrderItem, error) {
	// Convert product price to order currency using exchange rate
	unitPrice := product.BasePrice.Mul(exchangeRate)

	return NewOrderItem(
		orderID,
		product.ID,
		product.Name,
		product.SKU,
		unitPrice,
		quantity,
	)
}

// Validate checks if the OrderItem fields meet business rules.
func (oi *OrderItem) Validate() error {
	// Product ID validation
	if oi.ProductID == uuid.Nil {
		return ErrOrderItemProductIDRequired
	}

	// Product name validation (snapshot)
	if oi.ProductName == "" {
		return ErrOrderItemNameRequired
	}

	// Product SKU validation (snapshot)
	if oi.ProductSKU == "" {
		return ErrOrderItemSKURequired
	}

	// Unit price validation
	if oi.UnitPrice.LessThanOrEqual(decimal.Zero) {
		return ErrOrderItemPriceInvalid
	}

	// Quantity validation
	if oi.Quantity < 1 {
		return ErrOrderItemQuantityInvalid
	}

	return nil
}

// CalculateSubtotal computes the subtotal (unit price × quantity).
func (oi *OrderItem) CalculateSubtotal() {
	oi.Subtotal = oi.UnitPrice.Mul(
		decimal.NewFromInt(int64(oi.Quantity)),
	)
}

// UpdateQuantity changes the quantity and recalculates the subtotal.
func (oi *OrderItem) UpdateQuantity(newQuantity int) error {
	if newQuantity < 1 {
		return ErrOrderItemQuantityInvalid
	}

	oi.Quantity = newQuantity
	oi.CalculateSubtotal()

	return nil
}

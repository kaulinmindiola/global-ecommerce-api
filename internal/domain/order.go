package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order represents a purchase transaction in the e-commerce platform.
// It captures the state at the time of purchase including exchange rates.
type Order struct {
	ID           string          `json:"id"`
	OrderNumber  string          `json:"order_number"` // Human-readable: ORD-2025-000123
	UserID       string          `json:"user_id"`
	Currency     string          `json:"currency"`      // Currency chosen by customer
	ExchangeRate decimal.Decimal `json:"exchange_rate"` // Snapshot for audit trail
	Items        []OrderItem     `json:"items"`
	Subtotal     decimal.Decimal `json:"subtotal"`
	TaxAmount    decimal.Decimal `json:"tax_amount"`
	ShippingCost decimal.Decimal `json:"shipping_cost"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
	Status       OrderStatus     `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	ConfirmedAt  *time.Time      `json:"confirmed_at,omitempty"`
	CancelledAt  *time.Time      `json:"cancelled_at,omitempty"`
}

// Order validation errors
var (
	ErrOrderUserIDRequired      = errors.New("order user ID is required")
	ErrOrderCurrencyInvalid     = errors.New("order currency must be a valid 3-letter code")
	ErrOrderExchangeRateInvalid = errors.New("order exchange rate must be greater than zero")
	ErrOrderItemsRequired       = errors.New("order must have at least one item")
	ErrOrderTotalInvalid        = errors.New("order total must be greater than zero")
	ErrOrderAlreadyConfirmed    = errors.New("order is already confirmed")
	ErrOrderAlreadyCancelled    = errors.New("order is already cancelled")
	ErrOrderCannotCancel        = errors.New("order cannot be cancelled in current status")
)

// NewOrder creates a new Order with validated fields.
func NewOrder(userID, currency string, exchangeRate decimal.Decimal, items []OrderItem) (*Order, error) {
	// Generate order number: ORD-YYYY-NNNNNN
	orderNumber := generateOrderNumber()

	order := &Order{
		ID:           uuid.New().String(),
		OrderNumber:  orderNumber,
		UserID:       userID,
		Currency:     strings.ToUpper(strings.TrimSpace(currency)),
		ExchangeRate: exchangeRate,
		Items:        items,
		Subtotal:     decimal.Zero,
		TaxAmount:    decimal.Zero,
		ShippingCost: decimal.Zero,
		TotalAmount:  decimal.Zero,
		Status:       OrderStatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// Calculate totals
	order.CalculateTotals()

	if err := order.Validate(); err != nil {
		return nil, err
	}

	return order, nil
}

// Validate checks if the Order fields meet business rules.
func (o *Order) Validate() error {
	// User ID validation
	if o.UserID == "" {
		return ErrOrderUserIDRequired
	}

	// Currency validation
	if !currencyRegex.MatchString(o.Currency) {
		return ErrOrderCurrencyInvalid
	}

	// Exchange rate validation
	if o.ExchangeRate.LessThanOrEqual(decimal.Zero) {
		return ErrOrderExchangeRateInvalid
	}

	// Items validation
	if len(o.Items) == 0 {
		return ErrOrderItemsRequired
	}

	// Total validation
	if o.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return ErrOrderTotalInvalid
	}

	return nil
}

// CalculateTotals computes subtotal, tax, shipping, and total amount.
func (o *Order) CalculateTotals() {
	// Calculate subtotal from all items
	subtotal := decimal.Zero
	for _, item := range o.Items {
		subtotal = subtotal.Add(item.Subtotal)
	}
	o.Subtotal = subtotal

	// Tax calculation (can be customized based on location)
	// For now, we don't apply tax (set to zero)
	o.TaxAmount = decimal.Zero

	// Shipping cost (can be calculated based on weight, location, etc.)
	// For now, we use a flat rate
	o.ShippingCost = calculateShippingCost(subtotal)

	// Total = Subtotal + Tax + Shipping
	o.TotalAmount = o.Subtotal.Add(o.TaxAmount).Add(o.ShippingCost)
	o.UpdatedAt = time.Now().UTC()
}

// Confirm transitions the order to confirmed status.
func (o *Order) Confirm() error {
	if o.Status == OrderStatusConfirmed {
		return ErrOrderAlreadyConfirmed
	}

	if o.Status == OrderStatusCancelled {
		return ErrOrderAlreadyCancelled
	}

	o.Status = OrderStatusConfirmed
	now := time.Now().UTC()
	o.ConfirmedAt = &now
	o.UpdatedAt = now

	return nil
}

// Cancel transitions the order to cancelled status.
// Only pending and confirmed orders can be cancelled.
func (o *Order) Cancel() error {
	if o.Status == OrderStatusCancelled {
		return ErrOrderAlreadyCancelled
	}

	// Can only cancel pending or confirmed orders
	if o.Status != OrderStatusPending && o.Status != OrderStatusConfirmed {
		return ErrOrderCannotCancel
	}

	o.Status = OrderStatusCancelled
	now := time.Now().UTC()
	o.CancelledAt = &now
	o.UpdatedAt = now

	return nil
}

// Ship transitions the order to shipped status.
func (o *Order) Ship() error {
	if o.Status != OrderStatusConfirmed {
		return errors.New("only confirmed orders can be shipped")
	}

	o.Status = OrderStatusShipped
	o.UpdatedAt = time.Now().UTC()

	return nil
}

// Deliver transitions the order to delivered status.
func (o *Order) Deliver() error {
	if o.Status != OrderStatusShipped {
		return errors.New("only shipped orders can be delivered")
	}

	o.Status = OrderStatusDelivered
	o.UpdatedAt = time.Now().UTC()

	return nil
}

// CanBeCancelled returns true if the order can be cancelled.
func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusConfirmed
}

// IsPending returns true if the order is pending.
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsConfirmed returns true if the order is confirmed.
func (o *Order) IsConfirmed() bool {
	return o.Status == OrderStatusConfirmed
}

// IsCancelled returns true if the order is cancelled.
func (o *Order) IsCancelled() bool {
	return o.Status == OrderStatusCancelled
}

// generateOrderNumber creates a unique order number.
// Format: ORD-YYYY-NNNNNN (e.g., ORD-2025-000123)
func generateOrderNumber() string {
	year := time.Now().Year()
	// In production, this would be a sequence from database
	randomPart := uuid.New().String()[:6]
	return fmt.Sprintf("ORD-%d-%s", year, strings.ToUpper(randomPart))
}

// calculateShippingCost calculates shipping cost based on subtotal.
// This is a simplified version; in production, it would consider weight, distance, etc.
func calculateShippingCost(subtotal decimal.Decimal) decimal.Decimal {
	// Free shipping for orders over $100
	threshold := decimal.NewFromFloat(100.00)
	if subtotal.GreaterThanOrEqual(threshold) {
		return decimal.Zero
	}

	// Flat rate shipping
	return decimal.NewFromFloat(15.00)
}

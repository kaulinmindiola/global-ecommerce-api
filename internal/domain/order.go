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

	order.CalculateTotals()

	if err := order.Validate(); err != nil {
		return nil, err
	}

	return order, nil
}

func (o *Order) Validate() error {
	if o.UserID == "" {
		return ErrOrderUserIDRequired
	}
	if len(o.Currency) != 3 {
		return ErrOrderCurrencyInvalid
	}
	if o.ExchangeRate.LessThanOrEqual(decimal.Zero) {
		return ErrOrderExchangeRateInvalid
	}
	if len(o.Items) == 0 {
		return ErrOrderItemsRequired
	}
	if o.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return ErrOrderTotalInvalid
	}
	return nil
}

func (o *Order) CalculateTotals() {
	subtotal := decimal.Zero
	for _, item := range o.Items {
		subtotal = subtotal.Add(item.Subtotal)
	}
	o.Subtotal = subtotal
	o.TaxAmount = decimal.Zero
	o.ShippingCost = calculateShippingCost(subtotal)
	o.TotalAmount = o.Subtotal.Add(o.TaxAmount).Add(o.ShippingCost)
	o.UpdatedAt = time.Now().UTC()
}

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

func (o *Order) Cancel() error {
	if o.Status == OrderStatusCancelled {
		return ErrOrderAlreadyCancelled
	}
	if o.Status != OrderStatusPending && o.Status != OrderStatusConfirmed {
		return ErrOrderCannotCancel
	}
	o.Status = OrderStatusCancelled
	now := time.Now().UTC()
	o.CancelledAt = &now
	o.UpdatedAt = now
	return nil
}

func (o *Order) Ship() error {
	if o.Status != OrderStatusConfirmed {
		return errors.New("only confirmed orders can be shipped")
	}
	o.Status = OrderStatusShipped
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (o *Order) Deliver() error {
	if o.Status != OrderStatusShipped {
		return errors.New("only shipped orders can be delivered")
	}
	o.Status = OrderStatusDelivered
	o.UpdatedAt = time.Now().UTC()
	return nil
}

func (o *Order) CanBeCancelled() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusConfirmed
}
func (o *Order) IsPending() bool   { return o.Status == OrderStatusPending }
func (o *Order) IsConfirmed() bool { return o.Status == OrderStatusConfirmed }
func (o *Order) IsCancelled() bool { return o.Status == OrderStatusCancelled }

func generateOrderNumber() string {
	year := time.Now().Year()
	randomPart := uuid.New().String()[:6]
	return fmt.Sprintf("ORD-%d-%s", year, strings.ToUpper(randomPart))
}

func calculateShippingCost(subtotal decimal.Decimal) decimal.Decimal {
	threshold := decimal.NewFromFloat(100.00)
	if subtotal.GreaterThanOrEqual(threshold) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(15.00)
}

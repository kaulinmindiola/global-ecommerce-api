package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewOrder_Success(t *testing.T) {
	items := []OrderItem{
		{
			ID:          "item1",
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    2,
			Subtotal:    decimal.NewFromFloat(100.00),
		},
	}

	order, err := NewOrder(
		"user123",
		"USD",
		decimal.NewFromFloat(1.0),
		items,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.UserID != "user123" {
		t.Errorf("expected user ID 'user123', got '%s'", order.UserID)
	}

	if order.Currency != "USD" {
		t.Errorf("expected currency 'USD', got '%s'", order.Currency)
	}

	if order.Status != OrderStatusPending {
		t.Errorf("expected status 'pending', got '%s'", order.Status)
	}

	if len(order.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(order.Items))
	}

	expectedSubtotal := decimal.NewFromFloat(100.00)
	if !order.Subtotal.Equal(expectedSubtotal) {
		t.Errorf("expected subtotal %v, got %v", expectedSubtotal, order.Subtotal)
	}
}

func TestNewOrder_NoItems(t *testing.T) {
	_, err := NewOrder(
		"user123",
		"USD",
		decimal.NewFromFloat(1.0),
		[]OrderItem{},
	)

	if err != ErrOrderItemsRequired {
		t.Errorf("expected %v, got %v", ErrOrderItemsRequired, err)
	}
}

func TestNewOrder_InvalidCurrency(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	_, err := NewOrder(
		"user123",
		"INVALID",
		decimal.NewFromFloat(1.0),
		items,
	)

	if err != ErrOrderCurrencyInvalid {
		t.Errorf("expected %v, got %v", ErrOrderCurrencyInvalid, err)
	}
}

func TestOrder_Confirm(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)

	err := order.Confirm()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != OrderStatusConfirmed {
		t.Errorf("expected status 'confirmed', got '%s'", order.Status)
	}

	if order.ConfirmedAt == nil {
		t.Error("expected ConfirmedAt to be set")
	}
}

func TestOrder_Cancel_FromPending(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)

	err := order.Cancel()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != OrderStatusCancelled {
		t.Errorf("expected status 'cancelled', got '%s'", order.Status)
	}

	if order.CancelledAt == nil {
		t.Error("expected CancelledAt to be set")
	}
}

func TestOrder_CannotCancelShipped(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)
	order.Confirm()
	order.Ship()

	err := order.Cancel()

	if err != ErrOrderCannotCancel {
		t.Errorf("expected %v, got %v", ErrOrderCannotCancel, err)
	}
}

func TestOrder_Ship(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)
	order.Confirm()

	err := order.Ship()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != OrderStatusShipped {
		t.Errorf("expected status 'shipped', got '%s'", order.Status)
	}
}

func TestOrder_Deliver(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)
	order.Confirm()
	order.Ship()

	err := order.Deliver()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if order.Status != OrderStatusDelivered {
		t.Errorf("expected status 'delivered', got '%s'", order.Status)
	}
}

func TestOrder_CalculateTotals_FreeShipping(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(150.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(150.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)

	// Subtotal > $100, so shipping should be free
	if !order.ShippingCost.IsZero() {
		t.Errorf("expected free shipping, got %v", order.ShippingCost)
	}
}

func TestOrder_CalculateTotals_WithShipping(t *testing.T) {
	items := []OrderItem{
		{
			ProductID:   "prod1",
			ProductName: "Product 1",
			ProductSKU:  "SKU-001",
			UnitPrice:   decimal.NewFromFloat(50.00),
			Quantity:    1,
			Subtotal:    decimal.NewFromFloat(50.00),
		},
	}

	order, _ := NewOrder("user123", "USD", decimal.NewFromFloat(1.0), items)

	// Subtotal < $100, so shipping should be $15
	expectedShipping := decimal.NewFromFloat(15.00)
	if !order.ShippingCost.Equal(expectedShipping) {
		t.Errorf("expected shipping %v, got %v", expectedShipping, order.ShippingCost)
	}

	// Total should be subtotal + shipping
	expectedTotal := decimal.NewFromFloat(65.00)
	if !order.TotalAmount.Equal(expectedTotal) {
		t.Errorf("expected total %v, got %v", expectedTotal, order.TotalAmount)
	}
}

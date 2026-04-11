package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewOrderItem_Success(t *testing.T) {
	item, err := NewOrderItem(
		"order123",
		"prod123",
		"Wireless Headphones",
		"AUDIO-WH-001",
		decimal.NewFromFloat(99.99),
		2,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if item.OrderID != "order123" {
		t.Errorf("expected order ID 'order123', got '%s'", item.OrderID)
	}

	if item.ProductID != "prod123" {
		t.Errorf("expected product ID 'prod123', got '%s'", item.ProductID)
	}

	if item.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", item.Quantity)
	}

	expectedSubtotal := decimal.NewFromFloat(199.98)
	if !item.Subtotal.Equal(expectedSubtotal) {
		t.Errorf("expected subtotal %v, got %v", expectedSubtotal, item.Subtotal)
	}
}

func TestNewOrderItem_InvalidQuantity(t *testing.T) {
	_, err := NewOrderItem(
		"order123",
		"prod123",
		"Product",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		0,
	)

	if err != ErrOrderItemQuantityInvalid {
		t.Errorf("expected %v, got %v", ErrOrderItemQuantityInvalid, err)
	}
}

func TestNewOrderItem_InvalidPrice(t *testing.T) {
	_, err := NewOrderItem(
		"order123",
		"prod123",
		"Product",
		"SKU-001",
		decimal.Zero,
		1,
	)

	if err != ErrOrderItemPriceInvalid {
		t.Errorf("expected %v, got %v", ErrOrderItemPriceInvalid, err)
	}
}

func TestNewOrderItemFromProduct_Success(t *testing.T) {
	product, _ := NewProduct(
		"Wireless Headphones",
		"Premium headphones",
		"AUDIO-WH-001",
		decimal.NewFromFloat(100.00),
		"USD",
		50,
	)

	// Exchange rate: 1 USD = 0.92 EUR
	exchangeRate := decimal.NewFromFloat(0.92)

	item, err := NewOrderItemFromProduct(
		"order123",
		product,
		2,
		exchangeRate,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Unit price should be $100 * 0.92 = €92
	expectedUnitPrice := decimal.NewFromFloat(92.00)
	if !item.UnitPrice.Equal(expectedUnitPrice) {
		t.Errorf("expected unit price %v, got %v", expectedUnitPrice, item.UnitPrice)
	}

	// Subtotal should be €92 * 2 = €184
	expectedSubtotal := decimal.NewFromFloat(184.00)
	if !item.Subtotal.Equal(expectedSubtotal) {
		t.Errorf("expected subtotal %v, got %v", expectedSubtotal, item.Subtotal)
	}

	if item.ProductName != product.Name {
		t.Errorf("expected product name '%s', got '%s'", product.Name, item.ProductName)
	}

	if item.ProductSKU != product.SKU {
		t.Errorf("expected product SKU '%s', got '%s'", product.SKU, item.ProductSKU)
	}
}

func TestOrderItem_UpdateQuantity(t *testing.T) {
	item, _ := NewOrderItem(
		"order123",
		"prod123",
		"Product",
		"SKU-001",
		decimal.NewFromFloat(50.00),
		2,
	)

	err := item.UpdateQuantity(5)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if item.Quantity != 5 {
		t.Errorf("expected quantity 5, got %d", item.Quantity)
	}

	expectedSubtotal := decimal.NewFromFloat(250.00)
	if !item.Subtotal.Equal(expectedSubtotal) {
		t.Errorf("expected subtotal %v, got %v", expectedSubtotal, item.Subtotal)
	}
}

func TestOrderItem_CalculateSubtotal(t *testing.T) {
	tests := []struct {
		name      string
		unitPrice float64
		quantity  int
		want      float64
	}{
		{"simple calculation", 10.00, 2, 20.00},
		{"decimal price", 9.99, 3, 29.97},
		{"large quantity", 5.50, 100, 550.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, _ := NewOrderItem(
				"order123",
				"prod123",
				"Product",
				"SKU-001",
				decimal.NewFromFloat(tt.unitPrice),
				tt.quantity,
			)

			expected := decimal.NewFromFloat(tt.want)
			if !item.Subtotal.Equal(expected) {
				t.Errorf("expected subtotal %v, got %v", expected, item.Subtotal)
			}
		})
	}
}

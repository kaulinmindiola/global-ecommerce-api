package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewProduct_Success(t *testing.T) {
	price := decimal.NewFromFloat(99.99)
	product, err := NewProduct(
		"Wireless Headphones",
		"Premium noise-canceling headphones",
		"AUDIO-WH-001",
		price,
		"USD",
		50,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if product.Name != "Wireless Headphones" {
		t.Errorf("expected name 'Wireless Headphones', got '%s'", product.Name)
	}

	if product.SKU != "AUDIO-WH-001" {
		t.Errorf("expected SKU 'AUDIO-WH-001', got '%s'", product.SKU)
	}

	if !product.BasePrice.Equal(price) {
		t.Errorf("expected price %v, got %v", price, product.BasePrice)
	}

	if product.BaseCurrency != "USD" {
		t.Errorf("expected currency 'USD', got '%s'", product.BaseCurrency)
	}

	if product.StockQty != 50 {
		t.Errorf("expected stock 50, got %d", product.StockQty)
	}

	if !product.IsActive {
		t.Error("expected product to be active")
	}
}

func TestNewProduct_InvalidName(t *testing.T) {
	tests := []struct {
		name     string
		prodName string
		wantErr  error
	}{
		{"empty name", "", ErrProductNameRequired},
		{"too short", "A", ErrProductNameTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProduct(
				tt.prodName,
				"Description",
				"SKU-001",
				decimal.NewFromFloat(10.00),
				"USD",
				10,
			)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewProduct_InvalidSKU(t *testing.T) {
	tests := []struct {
		name string
		sku  string
	}{
		{"empty SKU", ""},
		{"invalid characters", "SKU_001"},
		{"lowercase", "sku-001"},
		{"special chars", "SKU@001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProduct(
				"Product Name",
				"Description",
				tt.sku,
				decimal.NewFromFloat(10.00),
				"USD",
				10,
			)

			if err == nil {
				t.Error("expected error for invalid SKU, got nil")
			}
		})
	}
}

func TestNewProduct_InvalidPrice(t *testing.T) {
	tests := []struct {
		name  string
		price decimal.Decimal
	}{
		{"zero price", decimal.Zero},
		{"negative price", decimal.NewFromFloat(-10.00)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProduct(
				"Product Name",
				"Description",
				"SKU-001",
				tt.price,
				"USD",
				10,
			)

			if err != ErrProductPriceInvalid {
				t.Errorf("expected %v, got %v", ErrProductPriceInvalid, err)
			}
		})
	}
}

func TestNewProduct_NegativeStock(t *testing.T) {
	_, err := NewProduct(
		"Product Name",
		"Description",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		"USD",
		-5,
	)

	if err != ErrProductStockNegative {
		t.Errorf("expected %v, got %v", ErrProductStockNegative, err)
	}
}

func TestProduct_ReserveStock_Success(t *testing.T) {
	product, _ := NewProduct(
		"Product",
		"Description",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		"USD",
		50,
	)

	err := product.ReserveStock(10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if product.StockQty != 40 {
		t.Errorf("expected stock 40, got %d", product.StockQty)
	}
}

func TestProduct_ReserveStock_InsufficientStock(t *testing.T) {
	product, _ := NewProduct(
		"Product",
		"Description",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		"USD",
		5,
	)

	err := product.ReserveStock(10)

	if err != ErrInsufficientStock {
		t.Errorf("expected %v, got %v", ErrInsufficientStock, err)
	}

	if product.StockQty != 5 {
		t.Errorf("expected stock unchanged at 5, got %d", product.StockQty)
	}
}

func TestProduct_RestoreStock(t *testing.T) {
	product, _ := NewProduct(
		"Product",
		"Description",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		"USD",
		40,
	)

	err := product.RestoreStock(10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if product.StockQty != 50 {
		t.Errorf("expected stock 50, got %d", product.StockQty)
	}
}

func TestProduct_UpdatePrice(t *testing.T) {
	product, _ := NewProduct(
		"Product",
		"Description",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		"USD",
		50,
	)

	newPrice := decimal.NewFromFloat(15.99)
	err := product.UpdatePrice(newPrice)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !product.BasePrice.Equal(newPrice) {
		t.Errorf("expected price %v, got %v", newPrice, product.BasePrice)
	}
}

func TestProduct_IsInStock(t *testing.T) {
	tests := []struct {
		name     string
		stockQty int
		want     bool
	}{
		{"in stock", 10, true},
		{"out of stock", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, _ := NewProduct(
				"Product",
				"Description",
				"SKU-001",
				decimal.NewFromFloat(10.00),
				"USD",
				tt.stockQty,
			)

			got := product.IsInStock()
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestProduct_CanFulfillOrder(t *testing.T) {
	product, _ := NewProduct(
		"Product",
		"Description",
		"SKU-001",
		decimal.NewFromFloat(10.00),
		"USD",
		20,
	)

	tests := []struct {
		name     string
		quantity int
		want     bool
	}{
		{"can fulfill", 10, true},
		{"exact stock", 20, true},
		{"cannot fulfill", 25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := product.CanFulfillOrder(tt.quantity)
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

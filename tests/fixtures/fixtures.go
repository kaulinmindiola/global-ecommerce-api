// Package fixtures provides pre-built domain objects for use in tests.
// All fixtures are deterministic — they use fixed UUIDs so assertions
// can reference specific IDs without generating them at test time.
package fixtures

import (
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/shopspring/decimal"
)

// ── Deterministic IDs ─────────────────────────────────────────────────────────

var (
	UserID    = uuid.MustParse("10000000-0000-4000-a000-000000000001")
	ProductID = uuid.MustParse("20000000-0000-4000-a000-000000000001")
	OrderID   = uuid.MustParse("30000000-0000-4000-a000-000000000001")
	USDID     = uuid.MustParse("40000000-0000-4000-a000-000000000001")
	EURID     = uuid.MustParse("40000000-0000-4000-a000-000000000002")
	COPID     = uuid.MustParse("40000000-0000-4000-a000-000000000008")
)

// ── Currency fixtures ─────────────────────────────────────────────────────────

// USD returns the US Dollar currency fixture.
func USD() *domain.Currency {
	return &domain.Currency{
		ID:                USDID,
		Code:              "USD",
		Name:              "US Dollar",
		Symbol:            "$",
		ExchangeRateToUSD: 1.0,
		IsActive:          true,
		RateUpdatedAt:     time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// EUR returns the Euro currency fixture.
func EUR() *domain.Currency {
	return &domain.Currency{
		ID:                EURID,
		Code:              "EUR",
		Name:              "Euro",
		Symbol:            "€",
		ExchangeRateToUSD: 0.925,
		IsActive:          true,
		RateUpdatedAt:     time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// COP returns the Colombian Peso currency fixture.
func COP() *domain.Currency {
	return &domain.Currency{
		ID:                COPID,
		Code:              "COP",
		Name:              "Colombian Peso",
		Symbol:            "$",
		ExchangeRateToUSD: 3950.0,
		IsActive:          true,
		RateUpdatedAt:     time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// ActiveCurrencies returns all three currency fixtures as a slice.
func ActiveCurrencies() []*domain.Currency {
	return []*domain.Currency{USD(), EUR(), COP()}
}

// ── User fixtures ─────────────────────────────────────────────────────────────

// User returns a valid active user fixture.
// The password hash corresponds to "SecurePass123!" with bcrypt cost 10.
func User() *domain.User {
	return &domain.User{
		ID:                  UserID,
		Email:               "kaulin@example.com",
		FullName:            "Kaulin Mindiola",
		PasswordHash:        "$2a$10$C8.11T9G4fW98XJ2x57N6.v547L1U3/n.s71f0gB0y0/w.1tH6eG2",
		PreferredCurrencyID: COPID,
		PreferredTimezone:   "America/Bogota",
		IsActive:            true,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
}

// ── Product fixtures ──────────────────────────────────────────────────────────

// Product returns a valid in-stock product fixture priced in USD.
func Product() *domain.Product {
	return &domain.Product{
		ID:               ProductID,
		Name:             "Wireless Bluetooth Headphones",
		Description:      "Premium noise-canceling headphones",
		SKU:              "AUDIO-WH-001",
		BasePrice:        decimal.NewFromFloat(149.99),
		BaseCurrencyID:   USDID,
		BaseCurrencyCode: "USD",
		StockQuantity:    45,
		IsActive:         true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}

// OutOfStockProduct returns a product with zero stock for testing edge cases.
func OutOfStockProduct() *domain.Product {
	p := Product()
	p.StockQuantity = 0
	p.SKU = "DESK-CONV-001"
	return p
}

// ── Order fixtures ────────────────────────────────────────────────────────────

// Order returns a valid pending order fixture.
func Order() *domain.Order {
	items := []domain.OrderItem{
		{
			ID:          uuid.New(),
			OrderID:     OrderID,
			ProductID:   ProductID,
			ProductName: "Wireless Bluetooth Headphones",
			ProductSKU:  "AUDIO-WH-001",
			UnitPrice:   decimal.NewFromFloat(138.74),
			Quantity:    2,
			Subtotal:    decimal.NewFromFloat(277.48),
			CreatedAt:   time.Now().UTC(),
		},
	}

	return &domain.Order{
		ID:           OrderID.String(),
		OrderNumber:  "ORD-2025-000001",
		UserID:       UserID.String(),
		Currency:     "EUR",
		ExchangeRate: decimal.NewFromFloat(0.925),
		Items:        items,
		Subtotal:     decimal.NewFromFloat(277.48),
		TaxAmount:    decimal.NewFromFloat(0),
		ShippingCost: decimal.NewFromFloat(15.00),
		TotalAmount:  decimal.NewFromFloat(292.48),
		Status:       domain.OrderStatusPending,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

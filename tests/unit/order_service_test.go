package unit_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/fixtures"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/mocks"
	"github.com/shopspring/decimal"
)

// setupOrderServiceMocks wires a full set of mocks with realistic fixture data.
func setupOrderServiceMocks() *mocks.AllMocks {
	m := mocks.NewAllMocks()

	// User returns the fixture user for any UUID.
	m.User.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
		return fixtures.User(), nil
	}

	// Currency returns fixture currencies by code.
	m.Currency.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
		switch code {
		case "USD":
			return fixtures.USD(), nil
		case "EUR":
			return fixtures.EUR(), nil
		case "COP":
			return fixtures.COP(), nil
		}
		return nil, domain.ErrNotFound
	}

	// Product returns the fixture product for any UUID.
	m.Product.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
		return fixtures.Product(), nil
	}

	// Order create succeeds by default.
	m.Order.CreateFn = func(_ context.Context, _ *domain.Order) error {
		return nil
	}

	return m
}

func TestOrderService_Create_Success(t *testing.T) {
	m := setupOrderServiceMocks()

	currSvc := service.NewCurrencyService(m.Currency, m.Cache)
	svc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

	req := service.CreateOrderRequest{
		UserID:   fixtures.UserID,
		Currency: "EUR",
		Items: []service.OrderItemRequest{
			{ProductID: fixtures.ProductID, Quantity: 2},
		},
	}

	resp, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.OrderNumber == "" {
		t.Error("expected order number to be set")
	}
	if resp.Currency != "EUR" {
		t.Errorf("currency: got %s, want EUR", resp.Currency)
	}
	if resp.Status != domain.OrderStatusPending {
		t.Errorf("status: got %s, want pending", resp.Status)
	}
	if len(resp.Items) != 1 {
		t.Errorf("items count: got %d, want 1", len(resp.Items))
	}
	if resp.TotalAmount.IsZero() {
		t.Error("total amount must not be zero")
	}
}

func TestOrderService_Create_InsufficientStock(t *testing.T) {
	m := setupOrderServiceMocks()

	// Override product to return out-of-stock item.
	m.Product.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
		p := fixtures.OutOfStockProduct()
		return p, nil
	}

	currSvc := service.NewCurrencyService(m.Currency, m.Cache)
	svc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

	req := service.CreateOrderRequest{
		UserID:   fixtures.UserID,
		Currency: "USD",
		Items:    []service.OrderItemRequest{{ProductID: fixtures.ProductID, Quantity: 1}},
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for out-of-stock product, got nil")
	}
}

func TestOrderService_Create_InvalidCurrency(t *testing.T) {
	m := setupOrderServiceMocks()

	// Override currency lookup to always fail.
	m.Currency.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
		return nil, domain.ErrNotFound
	}

	currSvc := service.NewCurrencyService(m.Currency, m.Cache)
	svc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

	req := service.CreateOrderRequest{
		UserID:   fixtures.UserID,
		Currency: "XYZ",
		Items:    []service.OrderItemRequest{{ProductID: fixtures.ProductID, Quantity: 1}},
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid currency, got nil")
	}
}

func TestOrderService_Create_EmptyItems(t *testing.T) {
	m := setupOrderServiceMocks()

	currSvc := service.NewCurrencyService(m.Currency, m.Cache)
	svc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

	req := service.CreateOrderRequest{
		UserID:   fixtures.UserID,
		Currency: "USD",
		Items:    []service.OrderItemRequest{}, // empty
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty items, got nil")
	}
}

func TestOrderService_Create_PriceConversionIsCorrect(t *testing.T) {
	m := setupOrderServiceMocks()

	// Product priced at 100 USD.
	m.Product.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
		p := fixtures.Product()
		p.BasePrice = decimal.NewFromFloat(100.0)
		p.BaseCurrencyCode = "USD"
		return p, nil
	}

	currSvc := service.NewCurrencyService(m.Currency, m.Cache)
	svc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

	req := service.CreateOrderRequest{
		UserID:   fixtures.UserID,
		Currency: "EUR",
		Items:    []service.OrderItemRequest{{ProductID: fixtures.ProductID, Quantity: 1}},
	}

	resp, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 100 USD * 0.925 (EUR rate) = 92.5 EUR
	expectedUnitPrice := decimal.NewFromFloat(92.5)
	if len(resp.Items) == 0 {
		t.Fatal("expected at least one item in response")
	}
	if !resp.Items[0].UnitPrice.Equal(expectedUnitPrice) {
		t.Errorf("unit price: got %s, want %s",
			resp.Items[0].UnitPrice.String(),
			expectedUnitPrice.String())
	}
}

func TestOrderService_UpdateStatus_ValidTransitions(t *testing.T) {
	transitions := []struct {
		name    string
		current domain.OrderStatus
		next    domain.OrderStatus
		wantErr bool
	}{
		{"pending to confirmed", domain.OrderStatusPending, domain.OrderStatusConfirmed, false},
		{"pending to cancelled", domain.OrderStatusPending, domain.OrderStatusCancelled, false},
		{"confirmed to shipped", domain.OrderStatusConfirmed, domain.OrderStatusShipped, false},
		{"confirmed to cancelled", domain.OrderStatusConfirmed, domain.OrderStatusCancelled, false},
		{"shipped to delivered", domain.OrderStatusShipped, domain.OrderStatusDelivered, false},
		{"delivered to cancelled (invalid)", domain.OrderStatusDelivered, domain.OrderStatusCancelled, true},
		{"cancelled to confirmed (invalid)", domain.OrderStatusCancelled, domain.OrderStatusConfirmed, true},
	}

	for _, tt := range transitions {
		t.Run(tt.name, func(t *testing.T) {
			m := setupOrderServiceMocks()

			// Return an order with the current status.
			order := fixtures.Order()
			order.Status = tt.current
			orderUUID := uuid.MustParse(order.ID)
			m.Order.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
				return order, nil
			}

			currSvc := service.NewCurrencyService(m.Currency, m.Cache)
			svc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

			err := svc.UpdateStatus(context.Background(), orderUUID, tt.next)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

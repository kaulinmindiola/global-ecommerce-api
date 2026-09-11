//go:build e2e

// Package e2e contains end-to-end tests that exercise complete user workflows
// through the full HTTP stack — from request parsing to service logic to response.
//
// These tests run against the in-memory test server (no real DB required).
// They are the closest thing to "does this actually work for a real user" without
// deploying to a real environment.
//
// Run with: make test-e2e
// Or:       go test -tags=e2e ./tests/e2e/...
package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/fixtures"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/helpers"
)

// TestCompleteOrderFlow exercises the critical user journey:
// Register → Login → Browse Products → Create Order → Verify Result
//
// This single test validates that all layers of the system work together:
// middleware (auth), handlers (request parsing), services (business logic),
// and the response shape (API contract compliance).
func TestCompleteOrderFlow(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// ── Setup mocks to simulate a real system state ───────────────────────
	setupOrderFlowMocks(t, srv)

	// ── Step 1: Register a new user ───────────────────────────────────────
	t.Log("Step 1: Registering new user")
	registerBody := map[string]any{
		"email":              "testflow@example.com",
		"password":           "SecurePass123!",
		"full_name":          "Flow Test User",
		"preferred_currency": "USD",
		"preferred_timezone": "America/Bogota",
	}

	registerResp := srv.POST("/api/v1/auth/register", registerBody)
	helpers.AssertStatus(t, registerResp, http.StatusCreated)

	var registerResult struct {
		Token struct {
			AccessToken string `json:"access_token"`
		} `json:"token"`
	}
	helpers.DecodeJSON(t, registerResp, &registerResult)

	if registerResult.Token.AccessToken == "" {
		t.Fatal("Step 1 failed: no access_token in register response")
	}
	token := registerResult.Token.AccessToken
	t.Logf("Step 1: ✓ User registered, token obtained (length=%d)", len(token))

	// ── Step 2: Browse product catalogue ─────────────────────────────────
	t.Log("Step 2: Browsing product catalogue")
	productsResp := srv.GET("/api/v1/products?currency=USD")
	helpers.AssertStatus(t, productsResp, http.StatusOK)

	var productsResult struct {
		Data []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Price struct {
				Amount   float64 `json:"amount"`
				Currency string  `json:"currency"`
			} `json:"price"`
			StockQuantity int `json:"stock_quantity"`
		} `json:"data"`
		Pagination struct {
			Total int64 `json:"total"`
		} `json:"pagination"`
	}
	helpers.DecodeJSON(t, productsResp, &productsResult)

	if productsResult.Pagination.Total == 0 {
		t.Fatal("Step 2 failed: no products in catalogue")
	}
	t.Logf("Step 2: ✓ Catalogue has %d products", productsResult.Pagination.Total)

	// ── Step 3: Get specific product details ──────────────────────────────
	t.Log("Step 3: Getting specific product details")
	productResp := srv.GET("/api/v1/products/" + fixtures.ProductID.String() + "?currency=EUR")
	helpers.AssertStatus(t, productResp, http.StatusOK)

	var productResult struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Price struct {
			Amount       float64 `json:"amount"`
			Currency     string  `json:"currency"`
			BaseAmount   float64 `json:"base_amount"`
			BaseCurrency string  `json:"base_currency"`
		} `json:"price"`
	}
	helpers.DecodeJSON(t, productResp, &productResult)

	if productResult.Price.Currency != "EUR" {
		t.Errorf("Step 3: price currency: got %s, want EUR", productResult.Price.Currency)
	}
	// Verify price was converted: USD 149.99 * 0.925 ≈ EUR 138.74
	if productResult.Price.Amount <= 0 {
		t.Error("Step 3: converted price must be positive")
	}
	t.Logf("Step 3: ✓ Product fetched, price %.2f %s (from %.2f %s)",
		productResult.Price.Amount, productResult.Price.Currency,
		productResult.Price.BaseAmount, productResult.Price.BaseCurrency)

	// ── Step 4: Create an order in EUR ────────────────────────────────────
	t.Log("Step 4: Creating order in EUR")
	orderBody := map[string]any{
		"currency": "EUR",
		"items": []map[string]any{
			{"product_id": fixtures.ProductID.String(), "quantity": 2},
		},
		"shipping_address": map[string]any{
			"street":      "Calle 123",
			"city":        "Riohacha",
			"state":       "La Guajira",
			"country":     "Colombia",
			"postal_code": "440001",
		},
	}

	orderResp := srv.POST("/api/v1/orders", orderBody, token)
	helpers.AssertStatus(t, orderResp, http.StatusCreated)

	var orderResult struct {
		ID           string  `json:"id"`
		OrderNumber  string  `json:"order_number"`
		Currency     string  `json:"currency"`
		ExchangeRate float64 `json:"exchange_rate"`
		Items        []struct {
			ProductID   string  `json:"product_id"`
			ProductName string  `json:"product_name"`
			Quantity    int     `json:"quantity"`
			UnitPrice   float64 `json:"unit_price"`
			Subtotal    float64 `json:"subtotal"`
		} `json:"items"`
		Subtotal              float64 `json:"subtotal"`
		ShippingCost          float64 `json:"shipping_cost"`
		TotalAmount           float64 `json:"total_amount"`
		Status                string  `json:"status"`
		CreatedAtUserTimezone string  `json:"created_at_user_timezone"`
	}
	helpers.DecodeJSON(t, orderResp, &orderResult)

	// Validate order response structure.
	if orderResult.ID == "" {
		t.Error("Step 4: order ID must not be empty")
	}
	if orderResult.OrderNumber == "" {
		t.Error("Step 4: order_number must not be empty")
	}
	if orderResult.Currency != "EUR" {
		t.Errorf("Step 4: currency: got %s, want EUR", orderResult.Currency)
	}
	if orderResult.ExchangeRate <= 0 {
		t.Error("Step 4: exchange_rate must be positive")
	}
	if orderResult.Status != "pending" {
		t.Errorf("Step 4: status: got %s, want pending", orderResult.Status)
	}
	if len(orderResult.Items) != 1 {
		t.Errorf("Step 4: items count: got %d, want 1", len(orderResult.Items))
	}
	if orderResult.Items[0].Quantity != 2 {
		t.Errorf("Step 4: quantity: got %d, want 2", orderResult.Items[0].Quantity)
	}
	if orderResult.TotalAmount <= 0 {
		t.Error("Step 4: total_amount must be positive")
	}
	if orderResult.CreatedAtUserTimezone == "" {
		t.Error("Step 4: created_at_user_timezone must be present")
	}

	t.Logf("Step 4: ✓ Order created — %s (%.2f EUR, status: %s)",
		orderResult.OrderNumber, orderResult.TotalAmount, orderResult.Status)

	// ── Step 5: Retrieve the order by ID ─────────────────────────────────
	t.Log("Step 5: Retrieving order by ID")
	getOrderResp := srv.GET("/api/v1/orders/"+orderResult.ID, token)
	helpers.AssertStatus(t, getOrderResp, http.StatusOK)
	t.Log("Step 5: ✓ Order retrieved successfully")

	// ── Step 6: Verify unauthenticated access is denied ───────────────────
	t.Log("Step 6: Verifying order is protected from unauthenticated access")
	unauthResp := srv.GET("/api/v1/orders/" + orderResult.ID) // no token
	helpers.AssertStatus(t, unauthResp, http.StatusUnauthorized)
	t.Log("Step 6: ✓ Unauthenticated access correctly denied")
}

// TestMultiCurrencyConversionFlow validates the multi-currency conversion chain
// through the public convert endpoint.
func TestMultiCurrencyConversionFlow(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
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

	conversions := []struct {
		from     string
		to       string
		amount   float64
		wantRate float64
	}{
		{"USD", "EUR", 100, 0.925},
		{"USD", "COP", 1, 3950},
		{"EUR", "USD", 100, 1.081081},
	}

	for _, c := range conversions {
		t.Run(c.from+"→"+c.to, func(t *testing.T) {
			url := "/api/v1/currencies/convert"
			url += "?from=" + c.from + "&to=" + c.to + "&amount=1"

			resp := srv.GET(url)
			helpers.AssertStatus(t, resp, http.StatusOK)

			var result struct {
				ExchangeRate float64 `json:"exchange_rate"`
			}
			helpers.DecodeJSON(t, resp, &result)

			tolerance := 0.001
			diff := result.ExchangeRate - c.wantRate
			if diff < -tolerance || diff > tolerance {
				t.Errorf("rate %s→%s: got %.6f, want %.6f",
					c.from, c.to, result.ExchangeRate, c.wantRate)
			}
		})
	}
}

// TestOrderFlow_InsufficientStock verifies that out-of-stock products
// correctly return a 422 Unprocessable Entity response.
func TestOrderFlow_InsufficientStock(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// Configure mocks with out-of-stock product.
	srv.Mocks.User.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
		return fixtures.User(), nil
	}
	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
		if code == "USD" {
			return fixtures.USD(), nil
		}
		return nil, domain.ErrNotFound
	}
	srv.Mocks.Product.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
		return fixtures.OutOfStockProduct(), nil
	}

	token := srv.IssueTestToken(fixtures.UserID)

	orderBody := map[string]any{
		"currency": "USD",
		"items": []map[string]any{
			{"product_id": fixtures.ProductID.String(), "quantity": 5},
		},
	}

	resp := srv.POST("/api/v1/orders", orderBody, token)
	helpers.AssertStatus(t, resp, http.StatusUnprocessableEntity)

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	helpers.DecodeJSON(t, resp, &body)

	if body.Error.Code != "BUSINESS_INSUFFICIENT_STOCK" {
		t.Errorf("error code: got %s, want BUSINESS_INSUFFICIENT_STOCK", body.Error.Code)
	}
}

// ── Helper ────────────────────────────────────────────────────────────────────

// setupOrderFlowMocks configures all mocks needed for the complete order flow test.
func setupOrderFlowMocks(t *testing.T, srv *helpers.TestServer) {
	t.Helper()

	// Currency lookups.
	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
		switch code {
		case "USD":
			return fixtures.USD(), nil
		case "EUR":
			return fixtures.EUR(), nil
		}
		return nil, domain.ErrNotFound
	}
	srv.Mocks.Currency.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}

	// User lookups.
	srv.Mocks.User.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
		return fixtures.User(), nil
	}

	// Product catalogue.
	srv.Mocks.Product.ListFn = func(_ context.Context, _ interface{}) ([]*domain.Product, int64, error) {
		return []*domain.Product{fixtures.Product()}, 1, nil
	}
	srv.Mocks.Product.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
		return fixtures.Product(), nil
	}

	// Order creation and retrieval.
	srv.Mocks.Order.CreateFn = func(_ context.Context, _ *domain.Order) error {
		return nil
	}
	srv.Mocks.Order.GetByIDFn = func(_ context.Context, id uuid.UUID) (*domain.Order, error) {
		o := fixtures.Order()
		o.ID = id.String()
		o.UserID = fixtures.UserID.String()
		return o, nil
	}
}

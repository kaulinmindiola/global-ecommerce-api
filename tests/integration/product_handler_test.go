//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/fixtures"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/helpers"
)

func TestListProducts_Public_Returns200(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// CAMBIO AQUÍ: Cambia interface{} por repository.ProductListParams
	srv.Mocks.Product.ListFn = func(_ context.Context, _ repository.ProductListParams) ([]*domain.Product, int64, error) {
		return []*domain.Product{fixtures.Product()}, 1, nil
	}

	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}

	// No auth token required — product listing is public.
	resp := srv.GET("/api/v1/products")
	helpers.AssertStatus(t, resp, http.StatusOK)
}

func TestGetProduct_NotFound_Returns404(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.Product.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
		return nil, domain.ErrNotFound
	}

	resp := srv.GET("/api/v1/products/" + fixtures.ProductID.String())
	helpers.AssertStatus(t, resp, http.StatusNotFound)

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	helpers.DecodeJSON(t, resp, &body)

	if body.Error.Code != "RESOURCE_NOT_FOUND" {
		t.Errorf("error code: got %s, want RESOURCE_NOT_FOUND", body.Error.Code)
	}
}

func TestGetProduct_InvalidUUID_Returns400(t *testing.T) {
	srv := helpers.NewTestServer(t)

	resp := srv.GET("/api/v1/products/not-a-uuid")
	helpers.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestCreateProduct_WithoutAuth_Returns401(t *testing.T) {
	srv := helpers.NewTestServer(t)

	body := map[string]any{
		"name":           "Test Product",
		"sku":            "TEST-001",
		"base_price":     99.99,
		"base_currency":  "USD",
		"stock_quantity": 10,
	}

	// No token provided.
	resp := srv.POST("/api/v1/products", body)
	helpers.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestCreateProduct_WithAuth_Success(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}

	token := srv.IssueTestToken(fixtures.UserID)
	body := map[string]any{
		"name":           "New Headphones",
		"description":    "Great sound quality",
		"sku":            "AUDIO-NEW-001",
		"base_price":     199.99,
		"base_currency":  "USD",
		"stock_quantity": 25,
	}

	resp := srv.POST("/api/v1/products", body, token)
	helpers.AssertStatus(t, resp, http.StatusCreated)
}

func TestCreateProduct_DuplicateSKU_Returns409(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}
	srv.Mocks.Product.CreateFn = func(_ context.Context, _ *domain.Product) error {
		return domain.ErrConflict
	}

	token := srv.IssueTestToken(fixtures.UserID)
	body := map[string]any{
		"name":           "Duplicate Product",
		"sku":            "AUDIO-WH-001", // already exists in seed
		"base_price":     149.99,
		"base_currency":  "USD",
		"stock_quantity": 10,
	}

	resp := srv.POST("/api/v1/products", body, token)
	helpers.AssertStatus(t, resp, http.StatusConflict)
}

func TestDeleteProduct_WithAuth_Returns204(t *testing.T) {
	srv := helpers.NewTestServer(t)

	token := srv.IssueTestToken(fixtures.UserID)
	resp := srv.DELETE("/api/v1/products/"+fixtures.ProductID.String(), token)
	helpers.AssertStatus(t, resp, http.StatusNoContent)
}

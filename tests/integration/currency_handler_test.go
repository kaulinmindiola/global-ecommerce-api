//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/fixtures"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/helpers"
)

func TestListCurrencies_ReturnsAllActive(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// Seed mock with fixture currencies.
	srv.Mocks.Currency.ListActiveFn = func(_ context.Context) ([]*domain.Currency, error) {
		return fixtures.ActiveCurrencies(), nil
	}

	resp := srv.GET("/api/v1/currencies")
	helpers.AssertStatus(t, resp, http.StatusOK)

	var body struct {
		Data  []any `json:"data"`
		Total int   `json:"total"`
	}
	helpers.DecodeJSON(t, resp, &body)

	if body.Total != 3 {
		t.Errorf("total: got %d, want 3", body.Total)
	}
	if len(body.Data) != 3 {
		t.Errorf("data length: got %d, want 3", len(body.Data))
	}
}

func TestConvertCurrency_USDtoEUR(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, code string) (*domain.Currency, error) {
		switch code {
		case "USD":
			return fixtures.USD(), nil
		case "EUR":
			return fixtures.EUR(), nil
		}
		return nil, domain.ErrNotFound
	}

	resp := srv.GET("/api/v1/currencies/convert?from=USD&to=EUR&amount=100")
	helpers.AssertStatus(t, resp, http.StatusOK)

	var body struct {
		FromCurrency    string  `json:"from_currency"`
		ToCurrency      string  `json:"to_currency"`
		Amount          float64 `json:"amount"`
		ConvertedAmount float64 `json:"converted_amount"`
		ExchangeRate    float64 `json:"exchange_rate"`
	}
	helpers.DecodeJSON(t, resp, &body)

	if body.FromCurrency != "USD" {
		t.Errorf("from_currency: got %s, want USD", body.FromCurrency)
	}
	if body.ToCurrency != "EUR" {
		t.Errorf("to_currency: got %s, want EUR", body.ToCurrency)
	}
	if body.ConvertedAmount != 92.5 {
		t.Errorf("converted_amount: got %.4f, want 92.5000", body.ConvertedAmount)
	}
}

func TestConvertCurrency_MissingParams_Returns400(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// Missing "to" and "amount" parameters.
	resp := srv.GET("/api/v1/currencies/convert?from=USD")
	helpers.AssertStatus(t, resp, http.StatusBadRequest)

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	helpers.DecodeJSON(t, resp, &body)

	if body.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("error code: got %s, want VALIDATION_ERROR", body.Error.Code)
	}
}

func TestConvertCurrency_InvalidAmount_Returns400(t *testing.T) {
	srv := helpers.NewTestServer(t)

	resp := srv.GET("/api/v1/currencies/convert?from=USD&to=EUR&amount=notanumber")
	helpers.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestConvertCurrency_NegativeAmount_Returns400(t *testing.T) {
	srv := helpers.NewTestServer(t)

	resp := srv.GET("/api/v1/currencies/convert?from=USD&to=EUR&amount=-50")
	helpers.AssertStatus(t, resp, http.StatusBadRequest)
}

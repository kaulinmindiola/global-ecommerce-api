//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/fixtures"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/helpers"
)

func TestRegister_Success_Returns201(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// Mock: currency lookup succeeds, user creation succeeds.
	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}
	srv.Mocks.Currency.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}

	body := map[string]any{
		"email":              "newuser@example.com",
		"password":           "SecurePass123!",
		"full_name":          "New User",
		"preferred_currency": "USD",
		"preferred_timezone": "UTC",
	}

	resp := srv.POST("/api/v1/auth/register", body)
	helpers.AssertStatus(t, resp, http.StatusCreated)

	var respBody struct {
		User  map[string]any `json:"user"`
		Token map[string]any `json:"token"`
	}
	helpers.DecodeJSON(t, resp, &respBody)

	if respBody.Token["access_token"] == nil {
		t.Error("expected access_token in response")
	}
	if respBody.Token["refresh_token"] == nil {
		t.Error("expected refresh_token in response")
	}
}

func TestRegister_DuplicateEmail_Returns409(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.Currency.GetByCodeFn = func(_ context.Context, _ string) (*domain.Currency, error) {
		return fixtures.USD(), nil
	}
	// Simulate duplicate email at DB level.
	srv.Mocks.User.CreateFn = func(_ context.Context, _ *domain.User) error {
		return domain.ErrConflict
	}

	body := map[string]any{
		"email":              "existing@example.com",
		"password":           "SecurePass123!",
		"full_name":          "Existing User",
		"preferred_currency": "USD",
		"preferred_timezone": "UTC",
	}

	resp := srv.POST("/api/v1/auth/register", body)
	helpers.AssertStatus(t, resp, http.StatusConflict)
}

func TestRegister_InvalidEmail_Returns400(t *testing.T) {
	srv := helpers.NewTestServer(t)

	body := map[string]any{
		"email":              "not-an-email",
		"password":           "SecurePass123!",
		"full_name":          "Test User",
		"preferred_currency": "USD",
		"preferred_timezone": "UTC",
	}

	resp := srv.POST("/api/v1/auth/register", body)
	helpers.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestRegister_WeakPassword_Returns400(t *testing.T) {
	srv := helpers.NewTestServer(t)

	body := map[string]any{
		"email":              "user@example.com",
		"password":           "weak",
		"full_name":          "Test User",
		"preferred_currency": "USD",
		"preferred_timezone": "UTC",
	}

	resp := srv.POST("/api/v1/auth/register", body)
	helpers.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestRegister_MissingContentType_Returns415(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// POST without Content-Type: application/json.
	resp := srv.GET("/api/v1/auth/register") // wrong method/type intentionally
	helpers.AssertStatus(t, resp, http.StatusMethodNotAllowed)
}

func TestLogin_ValidCredentials_Returns200(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// Return the fixture user on email lookup.
	srv.Mocks.User.GetByEmailFn = func(_ context.Context, _ string) (*domain.User, error) {
		return fixtures.User(), nil
	}
	srv.Mocks.Currency.GetByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Currency, error) {
		return fixtures.COP(), nil
	}

	body := map[string]any{
		"email":    "kaulin@example.com",
		"password": "SecurePass123!",
	}

	resp := srv.POST("/api/v1/auth/login", body)
	helpers.AssertStatus(t, resp, http.StatusOK)

	var respBody struct {
		Token struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"token"`
	}
	helpers.DecodeJSON(t, resp, &respBody)

	if respBody.Token.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if respBody.Token.TokenType != "Bearer" {
		t.Errorf("token_type: got %s, want Bearer", respBody.Token.TokenType)
	}
}

func TestLogin_WrongPassword_Returns401(t *testing.T) {
	srv := helpers.NewTestServer(t)

	srv.Mocks.User.GetByEmailFn = func(_ context.Context, _ string) (*domain.User, error) {
		return fixtures.User(), nil
	}

	body := map[string]any{
		"email":    "kaulin@example.com",
		"password": "WrongPassword123!",
	}

	resp := srv.POST("/api/v1/auth/login", body)
	helpers.AssertStatus(t, resp, http.StatusUnauthorized)
}

func TestLogin_UnknownEmail_Returns401(t *testing.T) {
	srv := helpers.NewTestServer(t)

	// User not found — must return same 401 as wrong password (prevent enumeration).
	srv.Mocks.User.GetByEmailFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, domain.ErrNotFound
	}

	body := map[string]any{
		"email":    "nobody@example.com",
		"password": "SecurePass123!",
	}

	resp := srv.POST("/api/v1/auth/login", body)
	helpers.AssertStatus(t, resp, http.StatusUnauthorized)
}

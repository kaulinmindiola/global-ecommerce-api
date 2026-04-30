// Package helpers provides shared utilities for all test levels.
// It wires a full in-memory test server with mock repositories so tests run
// without a real PostgreSQL or Redis instance.
package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/handler"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/metrics"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
	"github.com/kaulinmindiola/global-ecommerce-api/tests/mocks"
)

// TestServer wraps httptest.Server and exposes convenience methods for
// making typed HTTP requests with automatic JSON (de)serialisation.
type TestServer struct {
	Server      *httptest.Server
	Mocks       *mocks.AllMocks
	AuthService service.AuthService
	t           *testing.T
}

// NewTestServer creates a fully wired test server using mock repositories.
// No real DB or Redis connections are made — all data comes from the mocks.
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	m := mocks.NewAllMocks()
	met := metrics.New()

	// Wire real services on top of mock repositories.
	cacheSvc := service.NewAuthService("test-secret-key-32-chars-long!!", m.Cache)
	currSvc := service.NewCurrencyService(m.Currency, m.Cache)
	userSvc := service.NewUserService(m.User, m.Currency, m.Cache, currSvc)
	productSvc := service.NewProductService(m.Product, m.Currency, m.Cache, currSvc)
	orderSvc := service.NewOrderService(m.Order, m.Product, m.User, currSvc, m.Cache)

	router := handler.NewRouter(handler.RouterDeps{
		UserService:     userSvc,
		AuthService:     cacheSvc,
		CurrencyService: currSvc,
		ProductService:  productSvc,
		OrderService:    orderSvc,
		DB:              nil, // health handler uses nil-safe probes in tests
		Redis:           nil,
		Metrics:         met,
		Version:         "test",
		BuildDate:       "test",
		CommitHash:      "test",
	})

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &TestServer{
		Server:      srv,
		Mocks:       m,
		AuthService: cacheSvc,
		t:           t,
	}
}

// ── Request helpers ───────────────────────────────────────────────────────────

// GET performs a GET request and returns the response.
func (ts *TestServer) GET(path string, token ...string) *http.Response {
	ts.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.url(path), nil)
	if err != nil {
		ts.t.Fatalf("building GET request: %v", err)
	}
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// POST performs a POST request with a JSON body and returns the response.
func (ts *TestServer) POST(path string, body any, token ...string) *http.Response {
	ts.t.Helper()
	return ts.doJSON(http.MethodPost, path, body, token...)
}

// PUT performs a PUT request with a JSON body and returns the response.
func (ts *TestServer) PUT(path string, body any, token ...string) *http.Response {
	ts.t.Helper()
	return ts.doJSON(http.MethodPut, path, body, token...)
}

// DELETE performs a DELETE request and returns the response.
func (ts *TestServer) DELETE(path string, token ...string) *http.Response {
	ts.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, ts.url(path), nil)
	if err != nil {
		ts.t.Fatalf("building DELETE request: %v", err)
	}
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("DELETE %s: %v", path, err)
	}
	return resp
}

// ── Response helpers ──────────────────────────────────────────────────────────

// DecodeJSON decodes the response body into dest and closes the body.
func DecodeJSON(t *testing.T, resp *http.Response, dest any) {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("decoding JSON: %v\nBody was: %s", err, body)
	}
}

// AssertStatus fails the test if the response status code is not expected.
func AssertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status %d, got %d\nBody: %s", expected, resp.StatusCode, body)
	}
}

// ── Auth helpers ──────────────────────────────────────────────────────────────

// IssueTestToken generates a valid JWT for a test user ID.
// Use this in tests that need an authenticated request.
func (ts *TestServer) IssueTestToken(userID uuid.UUID) string {
	ts.t.Helper()
	tokens, err := ts.AuthService.IssueTokens(context.Background(), userID)
	if err != nil {
		ts.t.Fatalf("issuing test token: %v", err)
	}
	return tokens.AccessToken
}

// ── Internal ─────────────────────────────────────────────────────────────────

func (ts *TestServer) url(path string) string {
	return ts.Server.URL + path
}

func (ts *TestServer) doJSON(method, path string, body any, token ...string) *http.Response {
	ts.t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		ts.t.Fatalf("marshalling request body: %v", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(), method, ts.url(path), bytes.NewReader(data),
	)
	if err != nil {
		ts.t.Fatalf("building %s request: %v", method, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if len(token) > 0 && token[0] != "" {
		req.Header.Set("Authorization", "Bearer "+token[0])
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

// ── Test data factories ───────────────────────────────────────────────────────

// NewTestUserID returns a deterministic UUID for test users.
func NewTestUserID() uuid.UUID {
	return uuid.MustParse("u1000000-0000-4000-a000-000000000001")
}

// NewTestProductID returns a deterministic UUID for test products.
func NewTestProductID() uuid.UUID {
	return uuid.MustParse("p1000000-0000-4000-a000-000000000001")
}

// NowUTC returns the current UTC time truncated to seconds (for comparison).
func NowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

// StringPtr returns a pointer to the given string — useful for optional fields.
func StringPtr(s string) *string { return &s }

// Float64Ptr returns a pointer to the given float64.
func Float64Ptr(f float64) *float64 { return &f }

// MustMarshal marshals v to JSON and panics on error. Test-only.
func MustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("MustMarshal: %v", err))
	}
	return b
}

// Package mocks provides in-memory mock implementations of all repository
// interfaces. Mocks allow unit and integration tests to run without a real
// database or Redis connection.
//
// Each mock implements the corresponding repository interface and provides
// configurable behaviour through function fields:
//
//	mock.User.CreateFn = func(ctx context.Context, u *domain.User) error {
//	    return domain.ErrConflict  // simulate duplicate email
//	}
package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
)

// AllMocks groups all mock repositories for easy injection.
type AllMocks struct {
	User     *MockUserRepository
	Product  *MockProductRepository
	Currency *MockCurrencyRepository
	Order    *MockOrderRepository
	Cache    *MockCacheRepository
}

// NewAllMocks creates all mocks with sensible no-op defaults.
func NewAllMocks() *AllMocks {
	return &AllMocks{
		User:     NewMockUserRepository(),
		Product:  NewMockProductRepository(),
		Currency: NewMockCurrencyRepository(),
		Order:    NewMockOrderRepository(),
		Cache:    NewMockCacheRepository(),
	}
}

// ── User Mock ─────────────────────────────────────────────────────────────────

type MockUserRepository struct {
	CreateFn     func(ctx context.Context, user *domain.User) error
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmailFn func(ctx context.Context, email string) (*domain.User, error)
	UpdateFn     func(ctx context.Context, user *domain.User) error
	DeleteFn     func(ctx context.Context, id uuid.UUID) error
	ListFn       func(ctx context.Context, params repository.ListParams) ([]*domain.User, int64, error)
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		CreateFn:     func(_ context.Context, _ *domain.User) error { return nil },
		GetByIDFn:    func(_ context.Context, _ uuid.UUID) (*domain.User, error) { return nil, domain.ErrNotFound },
		GetByEmailFn: func(_ context.Context, _ string) (*domain.User, error) { return nil, domain.ErrNotFound },
		UpdateFn:     func(_ context.Context, _ *domain.User) error { return nil },
		DeleteFn:     func(_ context.Context, _ uuid.UUID) error { return nil },
		ListFn:       func(_ context.Context, _ repository.ListParams) ([]*domain.User, int64, error) { return nil, 0, nil },
	}
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	return m.CreateFn(ctx, user)
}
func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.GetByEmailFn(ctx, email)
}
func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	return m.UpdateFn(ctx, user)
}
func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockUserRepository) List(ctx context.Context, params repository.ListParams) ([]*domain.User, int64, error) {
	return m.ListFn(ctx, params)
}

// ── Product Mock ──────────────────────────────────────────────────────────────

type MockProductRepository struct {
	CreateFn      func(ctx context.Context, product *domain.Product) error
	GetByIDFn     func(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetBySKUFn    func(ctx context.Context, sku string) (*domain.Product, error)
	UpdateFn      func(ctx context.Context, product *domain.Product) error
	DeleteFn      func(ctx context.Context, id uuid.UUID) error
	ListFn        func(ctx context.Context, params repository.ProductListParams) ([]*domain.Product, int64, error)
	UpdateStockFn func(ctx context.Context, id uuid.UUID, delta int) error
}

func NewMockProductRepository() *MockProductRepository {
	return &MockProductRepository{
		CreateFn:   func(_ context.Context, _ *domain.Product) error { return nil },
		GetByIDFn:  func(_ context.Context, _ uuid.UUID) (*domain.Product, error) { return nil, domain.ErrNotFound },
		GetBySKUFn: func(_ context.Context, _ string) (*domain.Product, error) { return nil, domain.ErrNotFound },
		UpdateFn:   func(_ context.Context, _ *domain.Product) error { return nil },
		DeleteFn:   func(_ context.Context, _ uuid.UUID) error { return nil },
		ListFn: func(_ context.Context, _ repository.ProductListParams) ([]*domain.Product, int64, error) {
			return nil, 0, nil
		},
		UpdateStockFn: func(_ context.Context, _ uuid.UUID, _ int) error { return nil },
	}
}

func (m *MockProductRepository) Create(ctx context.Context, p *domain.Product) error {
	return m.CreateFn(ctx, p)
}
func (m *MockProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	return m.GetBySKUFn(ctx, sku)
}
func (m *MockProductRepository) Update(ctx context.Context, p *domain.Product) error {
	return m.UpdateFn(ctx, p)
}
func (m *MockProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}
func (m *MockProductRepository) List(ctx context.Context, params repository.ProductListParams) ([]*domain.Product, int64, error) {
	return m.ListFn(ctx, params)
}
func (m *MockProductRepository) UpdateStock(ctx context.Context, id uuid.UUID, delta int) error {
	return m.UpdateStockFn(ctx, id, delta)
}

// ── Currency Mock ─────────────────────────────────────────────────────────────

type MockCurrencyRepository struct {
	CreateFn             func(ctx context.Context, currency *domain.Currency) error
	GetByIDFn            func(ctx context.Context, id uuid.UUID) (*domain.Currency, error)
	GetByCodeFn          func(ctx context.Context, code string) (*domain.Currency, error)
	ListActiveFn         func(ctx context.Context) ([]*domain.Currency, error)
	UpdateFn             func(ctx context.Context, currency *domain.Currency) error
	UpdateExchangeRateFn func(ctx context.Context, id uuid.UUID, rate float64) error
}

func NewMockCurrencyRepository() *MockCurrencyRepository {
	return &MockCurrencyRepository{
		CreateFn:             func(_ context.Context, _ *domain.Currency) error { return nil },
		GetByIDFn:            func(_ context.Context, _ uuid.UUID) (*domain.Currency, error) { return nil, domain.ErrNotFound },
		GetByCodeFn:          func(_ context.Context, _ string) (*domain.Currency, error) { return nil, domain.ErrNotFound },
		ListActiveFn:         func(_ context.Context) ([]*domain.Currency, error) { return nil, nil },
		UpdateFn:             func(_ context.Context, _ *domain.Currency) error { return nil },
		UpdateExchangeRateFn: func(_ context.Context, _ uuid.UUID, _ float64) error { return nil },
	}
}

func (m *MockCurrencyRepository) Create(ctx context.Context, c *domain.Currency) error {
	return m.CreateFn(ctx, c)
}
func (m *MockCurrencyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Currency, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockCurrencyRepository) GetByCode(ctx context.Context, code string) (*domain.Currency, error) {
	return m.GetByCodeFn(ctx, code)
}
func (m *MockCurrencyRepository) ListActive(ctx context.Context) ([]*domain.Currency, error) {
	return m.ListActiveFn(ctx)
}
func (m *MockCurrencyRepository) Update(ctx context.Context, c *domain.Currency) error {
	return m.UpdateFn(ctx, c)
}
func (m *MockCurrencyRepository) UpdateExchangeRate(ctx context.Context, id uuid.UUID, rate float64) error {
	return m.UpdateExchangeRateFn(ctx, id, rate)
}

// ── Order Mock ────────────────────────────────────────────────────────────────

type MockOrderRepository struct {
	CreateFn           func(ctx context.Context, order *domain.Order) error
	GetByIDFn          func(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	GetByOrderNumberFn func(ctx context.Context, orderNumber string) (*domain.Order, error)
	GetByUserIDFn      func(ctx context.Context, userID uuid.UUID, params repository.ListParams) ([]*domain.Order, int64, error)
	UpdateStatusFn     func(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
	ListFn             func(ctx context.Context, params repository.OrderListParams) ([]*domain.Order, int64, error)
}

func NewMockOrderRepository() *MockOrderRepository {
	return &MockOrderRepository{
		CreateFn:           func(_ context.Context, _ *domain.Order) error { return nil },
		GetByIDFn:          func(_ context.Context, _ uuid.UUID) (*domain.Order, error) { return nil, domain.ErrNotFound },
		GetByOrderNumberFn: func(_ context.Context, _ string) (*domain.Order, error) { return nil, domain.ErrNotFound },
		GetByUserIDFn: func(_ context.Context, _ uuid.UUID, _ repository.ListParams) ([]*domain.Order, int64, error) {
			return nil, 0, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ domain.OrderStatus) error { return nil },
		ListFn: func(_ context.Context, _ repository.OrderListParams) ([]*domain.Order, int64, error) {
			return nil, 0, nil
		},
	}
}

func (m *MockOrderRepository) Create(ctx context.Context, order *domain.Order) error {
	return m.CreateFn(ctx, order)
}
func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return m.GetByIDFn(ctx, id)
}
func (m *MockOrderRepository) GetByOrderNumber(ctx context.Context, num string) (*domain.Order, error) {
	return m.GetByOrderNumberFn(ctx, num)
}
func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, params repository.ListParams) ([]*domain.Order, int64, error) {
	return m.GetByUserIDFn(ctx, userID, params)
}
func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	return m.UpdateStatusFn(ctx, id, status)
}
func (m *MockOrderRepository) List(ctx context.Context, params repository.OrderListParams) ([]*domain.Order, int64, error) {
	return m.ListFn(ctx, params)
}

// ── Cache Mock ────────────────────────────────────────────────────────────────

type MockCacheRepository struct {
	store map[string][]byte
}

func NewMockCacheRepository() *MockCacheRepository {
	return &MockCacheRepository{store: make(map[string][]byte)}
}

func (m *MockCacheRepository) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	return nil // no-op in tests; use a real implementation when needed
}
func (m *MockCacheRepository) Get(_ context.Context, _ string, _ interface{}) error {
	return domain.ErrCacheMiss
}
func (m *MockCacheRepository) Delete(_ context.Context, _ string) error             { return nil }
func (m *MockCacheRepository) DeleteByPattern(_ context.Context, _ string) error    { return nil }
func (m *MockCacheRepository) Exists(_ context.Context, _ string) (bool, error)     { return false, nil }
func (m *MockCacheRepository) Increment(_ context.Context, _ string) (int64, error) { return 0, nil }

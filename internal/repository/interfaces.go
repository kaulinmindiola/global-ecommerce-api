package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
)

// UserRepository defines the contract for user data access operations.
// Any struct implementing this interface can be used as the data source,
// enabling easy swapping between PostgreSQL, in-memory, or mock implementations.
type UserRepository interface {
	// Create persists a new user to the data store.
	Create(ctx context.Context, user *domain.User) error

	// GetByID retrieves a user by their unique identifier.
	// Returns domain.ErrNotFound if the user does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// GetByEmail retrieves a user by their email address.
	// Returns domain.ErrNotFound if no user matches the email.
	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update modifies an existing user's data.
	// Returns domain.ErrNotFound if the user does not exist.
	Update(ctx context.Context, user *domain.User) error

	// Delete performs a soft delete on a user record.
	// Returns domain.ErrNotFound if the user does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a paginated list of users.
	List(ctx context.Context, params ListParams) ([]*domain.User, int64, error)
}

// ProductRepository defines the contract for product data access operations.
type ProductRepository interface {
	// Create persists a new product to the data store.
	Create(ctx context.Context, product *domain.Product) error

	// GetByID retrieves a product by its unique identifier.
	// Returns domain.ErrNotFound if the product does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)

	// GetBySKU retrieves a product by its Stock Keeping Unit code.
	// Returns domain.ErrNotFound if no product matches the SKU.
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)

	// Update modifies an existing product's data.
	// Returns domain.ErrNotFound if the product does not exist.
	Update(ctx context.Context, product *domain.Product) error

	// Delete performs a soft delete on a product record.
	// Returns domain.ErrNotFound if the product does not exist.
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a paginated list of active products.
	List(ctx context.Context, params ProductListParams) ([]*domain.Product, int64, error)

	// UpdateStock atomically adjusts the stock quantity for a product.
	// A negative delta reduces stock; positive delta increases it.
	// Returns domain.ErrInsufficientStock if the resulting quantity would be negative.
	UpdateStock(ctx context.Context, id uuid.UUID, delta int) error
}

// CurrencyRepository defines the contract for currency data access operations.
type CurrencyRepository interface {
	// Create persists a new currency to the data store.
	Create(ctx context.Context, currency *domain.Currency) error

	// GetByID retrieves a currency by its unique identifier.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Currency, error)

	// GetByCode retrieves a currency by its ISO 4217 code (e.g., "USD", "EUR").
	// Returns domain.ErrNotFound if no currency matches the code.
	GetByCode(ctx context.Context, code string) (*domain.Currency, error)

	// ListActive retrieves all currencies currently enabled in the system.
	ListActive(ctx context.Context) ([]*domain.Currency, error)

	// Update modifies an existing currency's data.
	Update(ctx context.Context, currency *domain.Currency) error

	// UpdateExchangeRate sets a new exchange rate for a currency relative to USD.
	// The rate_updated_at timestamp is automatically set to the current UTC time.
	UpdateExchangeRate(ctx context.Context, id uuid.UUID, rate float64) error
}

// OrderRepository defines the contract for order data access operations.
// Order operations require transactional support to ensure financial consistency.
type OrderRepository interface {
	// Create persists a new order along with its items within a transaction.
	// This operation must be atomic: either the order and all items are saved,
	// or none of them are (preventing partial order states).
	Create(ctx context.Context, order *domain.Order) error

	// GetByID retrieves a complete order, including its line items.
	// Returns domain.ErrNotFound if the order does not exist.
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)

	// GetByOrderNumber retrieves an order by its human-readable order number.
	// Returns domain.ErrNotFound if no order matches.
	GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error)

	// GetByUserID retrieves a paginated list of orders for a specific user.
	GetByUserID(ctx context.Context, userID uuid.UUID, params ListParams) ([]*domain.Order, int64, error)

	// UpdateStatus transitions an order to a new status.
	// Returns domain.ErrInvalidTransition if the status change is not allowed.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error

	// List retrieves a paginated and filtered list of orders (admin use).
	List(ctx context.Context, params OrderListParams) ([]*domain.Order, int64, error)
}

// CacheRepository defines the contract for cache operations using Redis.
// All TTL (Time-To-Live) values are passed as time.Duration for type safety.
type CacheRepository interface {
	// Set stores a value in the cache with an expiration duration.
	// If ttl is 0, the key persists indefinitely.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Get retrieves a value from the cache by key.
	// Returns ErrCacheMiss if the key does not exist or has expired.
	Get(ctx context.Context, key string, dest interface{}) error

	// Delete removes a key from the cache.
	// Does not return an error if the key does not exist.
	Delete(ctx context.Context, key string) error

	// DeleteByPattern removes all keys matching a glob-style pattern.
	// Use with caution — this operation can be slow on large datasets.
	// Example pattern: "product:*" removes all product cache entries.
	DeleteByPattern(ctx context.Context, pattern string) error

	// Exists checks whether a key is present in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// Increment atomically increases a numeric counter stored at key.
	// Returns the new value after incrementing.
	Increment(ctx context.Context, key string) (int64, error)
}

// --- Supporting Types for Query Parameters ---

// ListParams contains common pagination and sorting parameters.
type ListParams struct {
	// Page is the 1-indexed page number.
	Page int

	// PageSize is the number of items per page (max 100).
	PageSize int

	// SortBy is the field name to sort by (e.g., "created_at", "email").
	SortBy string

	// SortOrder is either "asc" or "desc".
	SortOrder string
}

// ProductListParams extends ListParams with product-specific filters.
type ProductListParams struct {
	ListParams

	// Search filters products by name or SKU (case-insensitive).
	Search string

	// CurrencyCode filters products by their base currency (e.g., "USD").
	CurrencyCode string

	// MinPrice filters products with a base_price >= MinPrice.
	MinPrice *float64

	// MaxPrice filters products with a base_price <= MaxPrice.
	MaxPrice *float64

	// InStockOnly when true excludes products with stock_quantity <= 0.
	InStockOnly bool
}

// OrderListParams extends ListParams with order-specific filters.
type OrderListParams struct {
	ListParams

	// UserID filters orders by a specific user (optional).
	UserID *uuid.UUID

	// Status filters orders by their current status (optional).
	Status *domain.OrderStatus

	// FromDate filters orders created on or after this timestamp (UTC).
	FromDate *time.Time

	// ToDate filters orders created on or before this timestamp (UTC).
	ToDate *time.Time
}

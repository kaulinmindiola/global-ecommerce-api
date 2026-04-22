package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"github.com/shopspring/decimal"
)

// ProductService defines all business operations for the product catalogue.
type ProductService interface {
	// Create persists a new product after validating all business rules.
	Create(ctx context.Context, req CreateProductRequest) (*domain.Product, error)

	// GetByID retrieves a product, with its price optionally converted to a
	// target currency. If targetCurrency is empty, the base price is returned.
	GetByID(ctx context.Context, id uuid.UUID, targetCurrency string) (*ProductResponse, error)

	// List retrieves a paginated, filtered product catalogue.
	// Prices are returned both in base currency and in the requested display currency.
	List(ctx context.Context, params ListProductsParams) (*ProductListResponse, error)

	// Update modifies an existing product's mutable fields.
	Update(ctx context.Context, id uuid.UUID, req UpdateProductRequest) (*domain.Product, error)

	// Delete soft-deletes a product from the catalogue.
	Delete(ctx context.Context, id uuid.UUID) error
}

// --- Request / Response DTOs ---

// CreateProductRequest is the validated input for creating a product.
type CreateProductRequest struct {
	Name          string
	Description   string
	SKU           string
	BasePrice     float64
	BaseCurrency  string // ISO 4217 code, e.g. "USD"
	StockQuantity int
}

// UpdateProductRequest contains the fields that may be changed after creation.
// Only non-zero values are applied (partial update semantics).
type UpdateProductRequest struct {
	Name         string
	Description  string
	BasePrice    float64
	BaseCurrency string
}

// ListProductsParams is the full filter + pagination request for the catalogue.
type ListProductsParams struct {
	repository.ProductListParams
	DisplayCurrency string // Currency to display prices in (default: USD)
}

// PriceInfo carries a price in both its base currency and the requested display currency.
// This is the shape defined in the API contract (page 11 of the spec doc).
type PriceInfo struct {
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	BaseAmount   float64 `json:"base_amount"`
	BaseCurrency string  `json:"base_currency"`
}

// ProductResponse is the enriched response sent to clients.
// It adds the converted price on top of the domain entity.
type ProductResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SKU           string    `json:"sku"`
	Price         PriceInfo `json:"price"`
	StockQuantity int       `json:"stock_quantity"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

// ProductListResponse wraps paginated results with metadata.
type ProductListResponse struct {
	Data       []*ProductResponse `json:"data"`
	Pagination PaginationMeta     `json:"pagination"`
}

// PaginationMeta holds the pagination envelope returned in all list responses.
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// productService is the concrete implementation of ProductService.
type productService struct {
	productRepo  repository.ProductRepository
	currencyRepo repository.CurrencyRepository
	cache        repository.CacheRepository
	currency     CurrencyService
}

// NewProductService creates a new ProductService with all required dependencies.
func NewProductService(
	productRepo repository.ProductRepository,
	currencyRepo repository.CurrencyRepository,
	cache repository.CacheRepository,
	currency CurrencyService,
) ProductService {
	return &productService{
		productRepo:  productRepo,
		currencyRepo: currencyRepo,
		cache:        cache,
		currency:     currency,
	}
}

// Create validates and persists a new product.
// Business rules enforced:
//   - SKU must be unique (enforced at DB level, mapped to ErrConflict)
//   - BasePrice must be > 0
//   - BaseCurrency must exist in the system
//   - StockQuantity must be >= 0
func (s *productService) Create(ctx context.Context, req CreateProductRequest) (*domain.Product, error) {
	if err := validateCreateProduct(req); err != nil {
		return nil, err
	}

	// Resolve the currency ID from the ISO code.
	currency, err := s.currency.GetByCode(ctx, req.BaseCurrency)
	if err != nil {
		return nil, fmt.Errorf("resolving base currency %q: %w", req.BaseCurrency, err)
	}

	now := time.Now().UTC()
	product := &domain.Product{
		ID:               uuid.New(),
		Name:             strings.TrimSpace(req.Name),
		Description:      strings.TrimSpace(req.Description),
		SKU:              strings.ToUpper(strings.TrimSpace(req.SKU)),
		BasePrice:        decimal.NewFromFloat(req.BasePrice),
		BaseCurrencyID:   currency.ID,
		BaseCurrencyCode: currency.Code,
		StockQuantity:    req.StockQuantity,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, fmt.Errorf("creating product: %w", err)
	}

	// Invalidate catalogue cache so new product appears immediately.
	_ = s.cache.DeleteByPattern(ctx, "products:list:*")

	return product, nil
}

// GetByID fetches a product and enriches it with converted pricing.
// Cache key: "product:{id}:{displayCurrency}"
func (s *productService) GetByID(ctx context.Context, id uuid.UUID, targetCurrency string) (*ProductResponse, error) {
	if targetCurrency == "" {
		targetCurrency = "USD"
	}

	cacheKey := fmt.Sprintf("product:%s:%s", id, targetCurrency)

	var cached ProductResponse
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching product %s: %w", id, err)
	}

	response, err := s.toProductResponse(ctx, product, targetCurrency)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, cacheKey, response, 10*time.Minute)

	return response, nil
}

// List retrieves a paginated catalogue with converted prices.
// Cache key: "products:list:{page}:{pageSize}:{currency}:{search}:{inStock}"
func (s *productService) List(ctx context.Context, params ListProductsParams) (*ProductListResponse, error) {
	if params.DisplayCurrency == "" {
		params.DisplayCurrency = "USD"
	}

	products, total, err := s.productRepo.List(ctx, params.ProductListParams)
	if err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}

	responses := make([]*ProductResponse, 0, len(products))
	for _, p := range products {
		resp, err := s.toProductResponse(ctx, p, params.DisplayCurrency)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &ProductListResponse{
		Data: responses,
		Pagination: PaginationMeta{
			Page:       params.Page,
			Limit:      pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// Update modifies mutable product fields. Validates input before persisting.
func (s *productService) Update(ctx context.Context, id uuid.UUID, req UpdateProductRequest) (*domain.Product, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching product to update %s: %w", id, err)
	}

	// Apply partial updates — only non-zero values overwrite existing ones.
	if req.Name != "" {
		product.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		product.Description = strings.TrimSpace(req.Description)
	}
	if req.BasePrice > 0 {
		product.BasePrice = decimal.NewFromFloat(req.BasePrice)
	}
	if req.BaseCurrency != "" {
		currency, err := s.currency.GetByCode(ctx, req.BaseCurrency)
		if err != nil {
			return nil, fmt.Errorf("resolving new base currency %q: %w", req.BaseCurrency, err)
		}
		product.BaseCurrencyID = currency.ID
		product.BaseCurrencyCode = currency.Code
	}

	product.UpdatedAt = time.Now().UTC()

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("updating product %s: %w", id, err)
	}

	// Invalidate both the specific product cache and the catalogue listing cache.
	_ = s.cache.DeleteByPattern(ctx, fmt.Sprintf("product:%s:*", id))
	_ = s.cache.DeleteByPattern(ctx, "products:list:*")

	return product, nil
}

// Delete soft-deletes a product and removes it from all cache layers.
func (s *productService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.productRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting product %s: %w", id, err)
	}

	_ = s.cache.DeleteByPattern(ctx, fmt.Sprintf("product:%s:*", id))
	_ = s.cache.DeleteByPattern(ctx, "products:list:*")

	return nil
}

// toProductResponse enriches a domain.Product with a converted PriceInfo.
// toProductResponse enriches a domain.Product with a converted PriceInfo.
func (s *productService) toProductResponse(ctx context.Context, p *domain.Product, displayCurrency string) (*ProductResponse, error) {
	var convertedAmount float64

	// Extraemos el float64 de nuestro Decimal seguro
	basePriceFloat := p.BasePrice.InexactFloat64()

	if displayCurrency == p.BaseCurrencyCode {
		// No conversion needed.
		convertedAmount = basePriceFloat
	} else {
		rate, err := s.currency.GetExchangeRate(ctx, p.BaseCurrencyCode, displayCurrency)
		if err != nil {
			// Graceful degradation
			convertedAmount = basePriceFloat
			displayCurrency = p.BaseCurrencyCode
		} else {
			convertedAmount = s.currency.ConvertAmount(basePriceFloat, rate)
		}
	}

	return &ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		SKU:         p.SKU,
		Price: PriceInfo{
			Amount:       convertedAmount,
			Currency:     displayCurrency,
			BaseAmount:   basePriceFloat, // Asignamos el float extraído
			BaseCurrency: p.BaseCurrencyCode,
		},
		StockQuantity: p.StockQuantity,
		IsActive:      p.IsActive,
		CreatedAt:     p.CreatedAt,
	}, nil
}

// validateCreateProduct enforces business rules before any DB operation.
func validateCreateProduct(req CreateProductRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("product name is required: %w", domain.ErrInvalidInput)
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("product name exceeds 255 characters: %w", domain.ErrInvalidInput)
	}
	if strings.TrimSpace(req.SKU) == "" {
		return fmt.Errorf("product SKU is required: %w", domain.ErrInvalidInput)
	}
	if !isValidSKU(req.SKU) {
		return fmt.Errorf("product SKU contains invalid characters (alphanumeric and hyphens only): %w", domain.ErrInvalidInput)
	}
	if req.BasePrice <= 0 {
		return fmt.Errorf("base price must be greater than zero: %w", domain.ErrInvalidInput)
	}
	if req.StockQuantity < 0 {
		return fmt.Errorf("stock quantity cannot be negative: %w", domain.ErrInvalidInput)
	}
	if req.BaseCurrency == "" {
		return fmt.Errorf("base currency is required: %w", domain.ErrInvalidInput)
	}
	if len(req.BaseCurrency) != 3 {
		return fmt.Errorf("base currency must be a 3-letter ISO 4217 code: %w", domain.ErrInvalidInput)
	}
	return nil
}

// isValidSKU checks that a SKU contains only uppercase letters, digits, and hyphens.
func isValidSKU(sku string) bool {
	for _, r := range sku {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' {
			return false
		}
	}
	return true
}

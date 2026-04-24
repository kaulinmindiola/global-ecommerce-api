package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// ProductHandler handles all product catalogue HTTP endpoints.
// GET    /api/v1/products
// POST   /api/v1/products
// GET    /api/v1/products/{id}
// PUT    /api/v1/products/{id}
// DELETE /api/v1/products/{id}
type ProductHandler struct {
	productService service.ProductService
}

// NewProductHandler creates a new ProductHandler.
func NewProductHandler(productService service.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

// ─────────────────────────────────────────────
// GET /api/v1/products
// ─────────────────────────────────────────────

// ListProducts returns a paginated, filtered product catalogue.
// Query params: page, limit, currency, search, min_price, max_price, in_stock, sort, order
// Response: 200 OK — matches spec pages 10-11
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	currency := q.Get("currency")
	if currency == "" {
		currency = "USD"
	}

	params := service.ListProductsParams{
		ProductListParams: repository.ProductListParams{
			ListParams: repository.ListParams{
				Page:      page,
				PageSize:  limit,
				SortBy:    q.Get("sort"),
				SortOrder: q.Get("order"),
			},
			Search:       q.Get("search"),
			CurrencyCode: q.Get("currency"),
			InStockOnly:  q.Get("in_stock") == "true",
		},
		DisplayCurrency: currency,
	}

	// Optional price range filters.
	if minStr := q.Get("min_price"); minStr != "" {
		if val, err := strconv.ParseFloat(minStr, 64); err == nil {
			params.MinPrice = &val
		}
	}
	if maxStr := q.Get("max_price"); maxStr != "" {
		if val, err := strconv.ParseFloat(maxStr, 64); err == nil {
			params.MaxPrice = &val
		}
	}

	result, err := h.productService.List(r.Context(), params)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, result)
}

// ─────────────────────────────────────────────
// POST /api/v1/products
// ─────────────────────────────────────────────

// createProductRequest is the JSON body for product creation.
type createProductRequest struct {
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	SKU           string  `json:"sku"`
	BasePrice     float64 `json:"base_price"`
	BaseCurrency  string  `json:"base_currency"`
	StockQuantity int     `json:"stock_quantity"`
}

// CreateProduct persists a new product in the catalogue.
// Response: 201 Created — matches spec page 13-14
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	product, err := h.productService.Create(r.Context(), service.CreateProductRequest{
		Name:          req.Name,
		Description:   req.Description,
		SKU:           req.SKU,
		BasePrice:     req.BasePrice,
		BaseCurrency:  req.BaseCurrency,
		StockQuantity: req.StockQuantity,
	})
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusCreated, product)
}

// ─────────────────────────────────────────────
// GET /api/v1/products/{id}
// ─────────────────────────────────────────────

// GetProduct retrieves a single product by UUID.
// Query param: currency — display prices in this currency (default USD)
// Response: 200 OK
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	// 1. Corregimos los argumentos (añadimos 'w')
	// 2. Cambiamos 'err' por 'ok' porque parseUUIDParam devuelve un booleano (bool)
	productID, ok := parseUUIDParam(w, r, "id")

	// 3. Verificamos 'ok'. Si es false, la función ya envió el error al cliente.
	if !ok {
		return
	}

	currency := r.URL.Query().Get("currency")

	// 4. Aquí usamos ':=' para declarar 'err' por primera vez como un tipo error real
	product, err := h.productService.GetByID(r.Context(), productID, currency)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, product)
}

// ─────────────────────────────────────────────
// PUT /api/v1/products/{id}
// ─────────────────────────────────────────────

// updateProductRequest is the partial-update JSON body for product changes.
type updateProductRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	BasePrice    float64 `json:"base_price"`
	BaseCurrency string  `json:"base_currency"`
}

// UpdateProduct modifies mutable fields of an existing product.
// Response: 200 OK
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	// CAMBIO 1: Se agrega 'w' y se cambia 'err' por 'ok' (bool)
	productID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		// La función parseUUIDParam ya envió el error al cliente si ok es false
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	// CAMBIO 2: Aquí 'err' se declara por primera vez como tipo 'error'
	product, err := h.productService.Update(r.Context(), productID, service.UpdateProductRequest{
		Name:         req.Name,
		Description:  req.Description,
		BasePrice:    req.BasePrice,
		BaseCurrency: req.BaseCurrency,
	})
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, product)
}

// ─────────────────────────────────────────────
// DELETE /api/v1/products/{id}
// ─────────────────────────────────────────────

// DeleteProduct soft-deletes a product from the catalogue.
// Response: 204 No Content
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	// CAMBIO 1: Se agrega 'w' y se cambia 'err' por 'ok' (bool)
	productID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	// CAMBIO 2: 'err' es de tipo 'error' (funciona bien con nil y domainErrorResponse)
	if err := h.productService.Delete(r.Context(), productID); err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	noContentResponse(w)
}

// ─────────────────────────────────────────────
// Shared helpers
// ─────────────────────────────────────────────

// parseUUIDParam extracts and parses a chi URL parameter as a UUID.

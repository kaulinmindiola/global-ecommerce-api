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

// ListProducts godoc
// @Summary      List and search products
// @Description  Get a paginated, filtered, and sorted catalogue of products. Supports multi-currency pricing display.
// @Tags         Products
// @Produce      json
// @Param        page       query     int     false  "Page number (default: 1)" default(1)
// @Param        limit      query     int     false  "Items per page (max 100, default: 20)" default(20)
// @Param        currency   query     string  false  "Display prices converted to this ISO 4217 currency code" default(USD)
// @Param        search     query     string  false  "Search products by name or SKU"
// @Param        min_price  query     number  false  "Filter products with price greater than or equal to this amount"
// @Param        max_price  query     number  false  "Filter products with price less than or equal to this amount"
// @Param        in_stock   query     bool    false  "If true, only return products with stock > 0"
// @Param        sort       query     string  false  "Field to sort by (e.g., 'price', 'created_at')"
// @Param        order      query     string  false  "Sort direction ('asc' or 'desc')"
// @Success      200        {object}  service.ProductListResponse "Paginated product list"
// @Failure      400        {object}  ErrorResponse "Invalid query parameters"
// @Failure      500        {object}  ErrorResponse "Internal Server Error"
// @Router       /products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 || limit > 100 {
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
		if val, parseErr := strconv.ParseFloat(minStr, 64); parseErr == nil {
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
	Name          string  `json:"name" example:"Wireless Headphones"`
	Description   string  `json:"description" example:"Noise-cancelling over-ear headphones"`
	SKU           string  `json:"sku" example:"WH-1000XM4"`
	BasePrice     float64 `json:"base_price" example:"299.99"`
	BaseCurrency  string  `json:"base_currency" example:"USD"`
	StockQuantity int     `json:"stock_quantity" example:"150"`
}

// CreateProduct godoc
// @Summary      Create product (Admin)
// @Description  Creates a new product in the catalogue. Requires administrator privileges.
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createProductRequest  true  "Product creation payload"
// @Success      201      {object}  service.ProductResponse       "Product created successfully"
// @Failure      400      {object}  ErrorResponse                 "Validation Error (e.g., duplicate SKU, missing fields)"
// @Failure      401      {object}  ErrorResponse                 "Unauthorized"
// @Failure      403      {object}  ErrorResponse                 "Forbidden (Admin only)"
// @Failure      500      {object}  ErrorResponse                 "Internal Server Error"
// @Router       /products [post]
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

// GetProduct godoc
// @Summary      Get product details
// @Description  Retrieve complete details of a specific product by its UUID.
// @Tags         Products
// @Produce      json
// @Param        id        path      string  true   "Product UUID" format(uuid)
// @Param        currency  query     string  false  "Convert price to this currency (default: USD)" default(USD)
// @Success      200       {object}  service.ProductResponse "Product details"
// @Failure      400       {object}  ErrorResponse "Invalid UUID format"
// @Failure      404       {object}  ErrorResponse "Product not found"
// @Failure      500       {object}  ErrorResponse "Internal Server Error"
// @Router       /products/{id} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	productID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	currency := r.URL.Query().Get("currency")

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
	Name         string  `json:"name" example:"Updated Headphones"`
	Description  string  `json:"description" example:"Updated description text"`
	BasePrice    float64 `json:"base_price" example:"249.99"`
	BaseCurrency string  `json:"base_currency" example:"USD"`
}

// UpdateProduct godoc
// @Summary      Update product (Admin)
// @Description  Modifies mutable fields (name, description, price) of an existing product.
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string                        true  "Product UUID" format(uuid)
// @Param        request  body      handler.updateProductRequest  true  "Fields to update"
// @Success      200      {object}  service.ProductResponse       "Product updated successfully"
// @Failure      400      {object}  ErrorResponse                 "Validation Error"
// @Failure      401      {object}  ErrorResponse                 "Unauthorized"
// @Failure      403      {object}  ErrorResponse                 "Forbidden (Admin only)"
// @Failure      404      {object}  ErrorResponse                 "Product not found"
// @Failure      500      {object}  ErrorResponse                 "Internal Server Error"
// @Router       /products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

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

// DeleteProduct godoc
// @Summary      Delete product (Admin)
// @Description  Performs a soft-delete on a product, removing it from the public catalogue.
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Product UUID" format(uuid)
// @Success      204  "Product deleted successfully (No Content)"
// @Failure      400  {object}  ErrorResponse "Invalid UUID format"
// @Failure      401  {object}  ErrorResponse "Unauthorized"
// @Failure      403  {object}  ErrorResponse "Forbidden (Admin only)"
// @Failure      404  {object}  ErrorResponse "Product not found"
// @Failure      500  {object}  ErrorResponse "Internal Server Error"
// @Router       /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

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
// notice: Implementation omitted here to preserve your existing code structure.

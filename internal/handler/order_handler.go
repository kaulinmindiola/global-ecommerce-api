package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// OrderHandler handles all order lifecycle HTTP endpoints.
// POST /api/v1/orders
// GET  /api/v1/orders
// GET  /api/v1/orders/{id}
// GET  /api/v1/orders/number/{orderNumber}
// PUT  /api/v1/orders/{id}/status
type OrderHandler struct {
	orderService service.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// ─────────────────────────────────────────────
// POST /api/v1/orders
// ─────────────────────────────────────────────

// createOrderItemRequest is a single product line within a new order.
type createOrderItemRequest struct {
	ProductID string `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Quantity  int    `json:"quantity" example:"2"`
}

// shippingAddressRequest maps the shipping_address JSON object.
type shippingAddressRequest struct {
	Street     string `json:"street" example:"123 Tech Lane"`
	City       string `json:"city" example:"San Francisco"`
	State      string `json:"state" example:"CA"`
	Country    string `json:"country" example:"USA"`
	PostalCode string `json:"postal_code" example:"94105"`
}

// createOrderRequest is the full JSON body for POST /orders.
// Matches the contract defined on spec pages 15-16.
type createOrderRequest struct {
	Currency        string                   `json:"currency" example:"USD"`
	Items           []createOrderItemRequest `json:"items"`
	ShippingAddress shippingAddressRequest   `json:"shipping_address"`
}

// CreateOrder godoc
// @Summary      Create order
// @Description  Create a new order with multi-currency support. Validates stock and captures current exchange rates.
// @Tags         Orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request  body      handler.createOrderRequest  true  "Order payload including items and shipping address"
// @Success      201      {object}  service.OrderResponse       "Order created successfully"
// @Failure      400      {object}  ErrorResponse               "Validation Error (Invalid JSON, empty items, invalid UUIDs)"
// @Failure      401      {object}  ErrorResponse               "Unauthorized"
// @Failure      422      {object}  ErrorResponse               "Unprocessable Entity (e.g., Insufficient stock)"
// @Failure      500      {object}  ErrorResponse               "Internal Server Error"
// @Router       /orders [post]
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Invalid JSON request body")
		return
	}

	// Validate items are present before parsing product IDs.
	if len(req.Items) == 0 {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Order must contain at least one item",
			ErrorDetail{Field: "items", Message: "At least one item is required", Code: "REQUIRED"},
		)
		return
	}

	// Parse product UUIDs from the request, collecting all invalid ones.
	serviceItems := make([]service.OrderItemRequest, 0, len(req.Items))
	for i, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			errorResponse(w, r, http.StatusBadRequest,
				"VALIDATION_ERROR", "Invalid product_id format",
				ErrorDetail{
					Field:   "items[" + string(rune('0'+i)) + "].product_id",
					Message: "Must be a valid UUID",
					Code:    "INVALID_FORMAT",
				},
			)
			return
		}
		serviceItems = append(serviceItems, service.OrderItemRequest{
			ProductID: productID,
			Quantity:  item.Quantity,
		})
	}

	order, err := h.orderService.Create(r.Context(), service.CreateOrderRequest{
		UserID:   userID,
		Currency: req.Currency,
		Items:    serviceItems,
		ShippingAddress: service.ShippingAddress{
			Street:     req.ShippingAddress.Street,
			City:       req.ShippingAddress.City,
			State:      req.ShippingAddress.State,
			Country:    req.ShippingAddress.Country,
			PostalCode: req.ShippingAddress.PostalCode,
		},
	})
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusCreated, order)
}

// ─────────────────────────────────────────────
// GET /api/v1/orders
// ─────────────────────────────────────────────

// ListOrders godoc
// @Summary      List user orders
// @Description  Get a paginated list of the authenticated user's order history.
// @Tags         Orders
// @Security     BearerAuth
// @Produce      json
// @Param        page   query     int  false  "Page number" default(1)
// @Param        limit  query     int  false  "Items per page" default(20)
// @Param        sort   query     string false "Sort field" default(created_at)
// @Param        order  query     string false "Sort order (asc, desc)" default(desc)
// @Success      200    {object}  service.OrderListResponse "Paginated order list"
// @Failure      401    {object}  ErrorResponse             "Unauthorized"
// @Failure      500    {object}  ErrorResponse             "Internal Server Error"
// @Router       /orders [get]
func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	params := paginationParams(r)

	result, err := h.orderService.ListByUser(r.Context(), userID, params)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, result)
}

// ─────────────────────────────────────────────
// GET /api/v1/orders/{id}
// ─────────────────────────────────────────────

// GetOrder godoc
// @Summary      Get order details
// @Description  Retrieve complete details of a specific order by UUID. Enforces ownership (user can only see their own orders).
// @Tags         Orders
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Order UUID" format(uuid)
// @Success      200  {object}  service.OrderResponse "Order details"
// @Failure      400  {object}  ErrorResponse         "Invalid UUID format"
// @Failure      401  {object}  ErrorResponse         "Unauthorized"
// @Failure      404  {object}  ErrorResponse         "Order not found or belongs to another user"
// @Failure      500  {object}  ErrorResponse         "Internal Server Error"
// @Router       /orders/{id} [get]
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	orderID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	order, err := h.orderService.GetByID(r.Context(), orderID, userID)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, order)
}

// ─────────────────────────────────────────────
// GET /api/v1/orders/number/{orderNumber}
// ─────────────────────────────────────────────

// GetOrderByNumber godoc
// @Summary      Get order by number
// @Description  Retrieve an order by its human-readable tracking number (e.g., ORD-2025-000123).
// @Tags         Orders
// @Security     BearerAuth
// @Produce      json
// @Param        orderNumber  path      string  true  "Order Tracking Number"
// @Success      200          {object}  service.OrderResponse "Order details"
// @Failure      400          {object}  ErrorResponse         "Missing order number"
// @Failure      401          {object}  ErrorResponse         "Unauthorized"
// @Failure      404          {object}  ErrorResponse         "Order not found"
// @Failure      500          {object}  ErrorResponse         "Internal Server Error"
// @Router       /orders/number/{orderNumber} [get]
func (h *OrderHandler) GetOrderByNumber(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	orderNumber := chi.URLParam(r, "orderNumber")
	if orderNumber == "" {
		errorResponse(w, r, http.StatusBadRequest,
			"VALIDATION_ERROR", "Order number is required")
		return
	}

	order, err := h.orderService.GetByOrderNumber(r.Context(), orderNumber, userID)
	if err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, order)
}

// ─────────────────────────────────────────────
// PUT /api/v1/orders/{id}/status
// ─────────────────────────────────────────────

// updateStatusRequest is the JSON body for status transitions.
type updateStatusRequest struct {
	Status string `json:"status" example:"shipped" enums:"pending,confirmed,shipped,delivered,cancelled"`
}

// UpdateOrderStatus godoc
// @Summary      Update order status (Admin)
// @Description  Transitions an order to a new status (e.g., from confirmed to shipped).
// @Tags         Orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "Order UUID" format(uuid)
// @Param        request  body      handler.updateStatusRequest  true  "New status payload"
// @Success      200      {object}  map[string]string            "Success message"
// @Failure      400      {object}  ErrorResponse                "Validation Error (Invalid JSON or invalid state transition)"
// @Failure      401      {object}  ErrorResponse                "Unauthorized"
// @Failure      403      {object}  ErrorResponse                "Forbidden"
// @Failure      404      {object}  ErrorResponse                "Order not found"
// @Failure      500      {object}  ErrorResponse                "Internal Server Error"
// @Router       /orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON")
		return
	}

	if err := h.orderService.UpdateStatus(r.Context(), orderID, domain.OrderStatus(req.Status)); err != nil {
		domainErrorResponse(w, r, err)
		return
	}

	successResponse(w, http.StatusOK, map[string]string{"message": "Order status updated"})
}

// ─────────────────────────────────────────────
// Shared helper
// ─────────────────────────────────────────────

// paginationParams extracts page and limit from query string with safe defaults.
func paginationParams(r *http.Request) repository.ListParams {
	q := r.URL.Query()

	page := 1
	limit := 20

	if p := q.Get("page"); p != "" {
		if v, err := parseInt(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := q.Get("limit"); l != "" {
		if v, err := parseInt(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	return repository.ListParams{
		Page:      page,
		PageSize:  limit,
		SortBy:    q.Get("sort"),
		SortOrder: q.Get("order"),
	}
}

// parseInt is a lightweight helper to avoid repeated strconv.Atoi error handling.
func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &invalidIntError{}
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

type invalidIntError struct{}

func (e *invalidIntError) Error() string { return "invalid integer" }

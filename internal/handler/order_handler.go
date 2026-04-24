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
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// shippingAddressRequest maps the shipping_address JSON object.
type shippingAddressRequest struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
}

// createOrderRequest is the full JSON body for POST /orders.
// Matches the contract defined on spec pages 15-16.
type createOrderRequest struct {
	Currency        string                   `json:"currency"`
	Items           []createOrderItemRequest `json:"items"`
	ShippingAddress shippingAddressRequest   `json:"shipping_address"`
}

// CreateOrder processes a new order for the authenticated user.
// Response: 201 Created — this is the most critical endpoint in the system.
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

// ListOrders returns the authenticated user's paginated order history.
// Response: 200 OK
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

// GetOrder retrieves a single order by UUID.
// Enforces that the order belongs to the authenticated user.
// Response: 200 OK
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == uuid.Nil {
		errorResponse(w, r, http.StatusUnauthorized,
			"AUTH_TOKEN_MISSING", "Authentication required")
		return
	}

	// CAMBIO: Usamos 'ok' (bool) para parseUUIDParam.
	// Esta función ya envía el errorResponse internamente si falla.
	orderID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return // Salimos porque el error ya se envió al cliente
	}

	// CAMBIO: Ahora declaramos 'err' por primera vez como un tipo error real
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

// GetOrderByNumber retrieves an order by its human-readable order number.
// Response: 200 OK
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
	Status string `json:"status"`
}

// UpdateOrderStatus transitions an order to a new status.
// Valid statuses: confirmed, shipped, delivered, cancelled
// Response: 200 OK
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID, ok := parseUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON")
		return
	}

	// Aquí 'err' es nuevo y de tipo error, por lo que funciona correctamente
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

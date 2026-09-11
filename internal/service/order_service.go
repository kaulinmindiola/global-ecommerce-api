package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
	"github.com/shopspring/decimal"
)

// OrderService defines all business operations for order lifecycle management.
// This is the most critical service: it coordinates currency conversion,
// inventory validation, and financial calculations in a single atomic operation.
type OrderService interface {
	// Create processes a new order request. This is THE core endpoint:
	// validates stock, converts prices, snapshots the exchange rate,
	// and persists everything atomically via the repository transaction.
	Create(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error)

	// GetByID retrieves a complete order with all line items.
	// Enforces ownership — users can only view their own orders.
	GetByID(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*OrderResponse, error)

	// GetByOrderNumber retrieves an order by its human-readable identifier (e.g. ORD-2025-ABC123).
	GetByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uuid.UUID) (*OrderResponse, error)

	// ListByUser retrieves paginated orders for a specific user.
	ListByUser(ctx context.Context, userID uuid.UUID, params repository.ListParams) (*OrderListResponse, error)

	// UpdateStatus transitions an order through its lifecycle state machine.
	// Delegates to domain methods: Confirm, Cancel, Ship, Deliver.
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
}

// --- Request / Response DTOs ---

// CreateOrderRequest is the validated input DTO for order creation.
// Maps directly to the POST /api/v1/orders body defined in the API spec.
type CreateOrderRequest struct {
	UserID          uuid.UUID
	Currency        string // ISO 4217 code, e.g. "EUR"
	Items           []OrderItemRequest
	ShippingAddress ShippingAddress
}

// OrderItemRequest is a single line item in the order creation request.
type OrderItemRequest struct {
	ProductID uuid.UUID
	Quantity  int
}

// ShippingAddress holds the delivery destination details.
type ShippingAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
}

// OrderItemResponse is the enriched line item returned in the order response.
// Matches the shape defined on page 18 of the API contracts spec.
type OrderItemResponse struct {
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name"`
	ProductSKU  string          `json:"product_sku"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	Subtotal    decimal.Decimal `json:"subtotal"`
}

// OrderResponse is the full order representation returned to the client.
// ExchangeRate is included for financial auditability per the API spec.
type OrderResponse struct {
	ID                    uuid.UUID           `json:"id"`
	OrderNumber           string              `json:"order_number"`
	UserID                uuid.UUID           `json:"user_id"`
	Currency              string              `json:"currency"`
	ExchangeRate          decimal.Decimal     `json:"exchange_rate"`
	Items                 []OrderItemResponse `json:"items"`
	Subtotal              decimal.Decimal     `json:"subtotal"`
	TaxAmount             decimal.Decimal     `json:"tax_amount"`
	ShippingCost          decimal.Decimal     `json:"shipping_cost"`
	TotalAmount           decimal.Decimal     `json:"total_amount"`
	Status                domain.OrderStatus  `json:"status"`
	CreatedAt             time.Time           `json:"created_at"`
	CreatedAtUserTimezone string              `json:"created_at_user_timezone"`
}

// OrderListResponse wraps paginated order results with pagination metadata.
type OrderListResponse struct {
	Data       []*OrderResponse `json:"data"`
	Pagination PaginationMeta   `json:"pagination"`
}

// orderService is the concrete implementation of OrderService.
type orderService struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
	userRepo    repository.UserRepository
	currency    CurrencyService
	cache       repository.CacheRepository
}

// NewOrderService constructs an OrderService with all required dependencies injected.
func NewOrderService(
	orderRepo repository.OrderRepository,
	productRepo repository.ProductRepository,
	userRepo repository.UserRepository,
	currency CurrencyService,
	cache repository.CacheRepository,
) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		userRepo:    userRepo,
		currency:    currency,
		cache:       cache,
	}
}

// resolvedItem holds a product with its computed prices for a single order line.
// Defined at package level so Create and buildOrderResponse share the same type.
type resolvedItem struct {
	product   *domain.Product
	quantity  int
	unitPrice decimal.Decimal
	subtotal  decimal.Decimal
}

// Create processes a complete order creation request.
//
// Business logic pipeline (matches spec page 16):
//  1. Validate the request structure.
//  2. Verify the requesting user exists and get their timezone.
//  3. Validate the target currency is supported in our system.
//  4. For each item: fetch the product, validate stock availability.
//  5. Get the exchange rate: product.BaseCurrency → target currency.
//  6. Calculate unit price using decimal.Decimal to avoid float64 drift.
//  7. Build domain OrderItems via domain.NewOrderItem (validates each item).
//  8. Build domain.Order via domain.NewOrder (validates + calculates totals).
//  9. Snapshot USD→targetCurrency rate at the order header for audit compliance.
//
// 10. Persist atomically via repository transaction (order + items + stock decrement).
// 11. Invalidate the user's cached order list.
func (s *orderService) Create(ctx context.Context, req CreateOrderRequest) (*OrderResponse, error) {
	if err := validateCreateOrderRequest(req); err != nil {
		return nil, err
	}

	// Step 2: Verify user exists to obtain their preferred timezone.
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("validating user %s: %w", req.UserID, err)
	}

	// Step 3: Validate the target currency is supported.
	targetCurrency, err := s.currency.GetByCode(ctx, strings.ToUpper(req.Currency))
	if err != nil {
		return nil, fmt.Errorf("invalid order currency %q: %w", req.Currency, domain.ErrInvalidInput)
	}

	// Steps 4–6: Resolve each product and compute prices.
	// ALL validations run before any financial state changes (fail-fast).
	resolved := make([]resolvedItem, 0, len(req.Items))

	for _, itemReq := range req.Items {
		product, err := s.productRepo.GetByID(ctx, itemReq.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found: %w", itemReq.ProductID, err)
		}

		if product.StockQuantity < itemReq.Quantity {
			return nil, fmt.Errorf(
				"insufficient stock for %q: requested %d, available %d: %w",
				product.Name, itemReq.Quantity, product.StockQuantity,
				domain.ErrInsufficientStock,
			)
		}

		// Step 5: Exchange rate from product base currency to the order currency.
		rateFloat, err := s.currency.GetExchangeRate(ctx, product.BaseCurrencyCode, targetCurrency.Code)
		if err != nil {
			return nil, fmt.Errorf("exchange rate %s→%s: %w",
				product.BaseCurrencyCode, targetCurrency.Code, err)
		}

		// Step 6: Use decimal.Decimal for all monetary arithmetic.
		rateDecimal := decimal.NewFromFloat(rateFloat)
		unitPrice := product.BasePrice.Mul(rateDecimal).Round(4)
		subtotal := unitPrice.Mul(decimal.NewFromInt(int64(itemReq.Quantity))).Round(4)

		resolved = append(resolved, resolvedItem{
			product:   product,
			quantity:  itemReq.Quantity,
			unitPrice: unitPrice,
			subtotal:  subtotal,
		})
	}

	// Step 9: Snapshot USD→targetCurrency rate for the order header.
	exchangeRateFloat, err := s.currency.GetExchangeRate(ctx, "USD", targetCurrency.Code)
	if err != nil {
		return nil, fmt.Errorf("snapshotting exchange rate USD→%s: %w", targetCurrency.Code, err)
	}
	exchangeRateSnapshot := decimal.NewFromFloat(exchangeRateFloat).Round(6)

	// Step 7: Pre-generate the orderID so we can pass it to NewOrderItem.
	orderID := uuid.New()

	// Build domain OrderItems — domain.NewOrderItem validates each one.
	domainItems := make([]domain.OrderItem, 0, len(resolved))
	for _, ri := range resolved {
		item, err := domain.NewOrderItem(
			orderID,
			ri.product.ID,
			ri.product.Name,
			ri.product.SKU,
			ri.unitPrice,
			ri.quantity,
		)
		if err != nil {
			return nil, fmt.Errorf("building order item for product %s: %w", ri.product.ID, err)
		}
		domainItems = append(domainItems, *item)
	}

	// Step 8: domain.NewOrder runs Validate() + CalculateTotals() internally.
	// We need to set the ID we pre-generated to keep it consistent with the items.
	order, err := domain.NewOrder(
		req.UserID.String(),
		targetCurrency.Code,
		exchangeRateSnapshot,
		domainItems,
	)
	if err != nil {
		return nil, fmt.Errorf("building order: %w", err)
	}
	// Override the auto-generated ID with our pre-generated one so items match.
	order.ID = orderID.String()

	// Step 10: Persist atomically. The repository owns the DB transaction.
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("persisting order: %w", err)
	}

	// Step 11: Bust the user's order list cache.
	_ = s.cache.DeleteByPattern(ctx, fmt.Sprintf("orders:user:%s:*", req.UserID))

	return buildOrderResponse(order, resolved, user.PreferredTimezone), nil
}

// GetByID retrieves a complete order, enforcing ownership authorization.
func (s *orderService) GetByID(ctx context.Context, id uuid.UUID, requestingUserID uuid.UUID) (*OrderResponse, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching order %s: %w", id, err)
	}

	// Authorization: users can only access their own orders.
	if order.UserID != requestingUserID.String() {
		return nil, domain.ErrForbidden
	}

	return s.toOrderResponse(ctx, order)
}

// GetByOrderNumber retrieves an order by its human-readable identifier.
func (s *orderService) GetByOrderNumber(ctx context.Context, orderNumber string, requestingUserID uuid.UUID) (*OrderResponse, error) {
	order, err := s.orderRepo.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching order %s: %w", orderNumber, err)
	}

	if order.UserID != requestingUserID.String() {
		return nil, domain.ErrForbidden
	}

	return s.toOrderResponse(ctx, order)
}

// ListByUser retrieves paginated orders for a specific user.
func (s *orderService) ListByUser(ctx context.Context, userID uuid.UUID, params repository.ListParams) (*OrderListResponse, error) {
	orders, total, err := s.orderRepo.GetByUserID(ctx, userID, params)
	if err != nil {
		return nil, fmt.Errorf("listing orders for user %s: %w", userID, err)
	}

	responses := make([]*OrderResponse, 0, len(orders))
	for _, o := range orders {
		resp, err := s.toOrderResponse(ctx, o)
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

	return &OrderListResponse{
		Data: responses,
		Pagination: PaginationMeta{
			Page:       params.Page,
			Limit:      pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateStatus transitions an order's status.
// Delegates each transition to the domain entity methods so the state machine
// lives in the domain layer where it belongs (Clean Architecture principle).
func (s *orderService) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	newStatus domain.OrderStatus,
) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching order %s: %w", id, err)
	}

	// Valid transitions are enforced by domain methods.
	switch newStatus {
	case domain.OrderStatusConfirmed:
		err = order.Confirm()

	case domain.OrderStatusCancelled:
		err = order.Cancel()

	case domain.OrderStatusShipped:
		err = order.Ship()

	case domain.OrderStatusDelivered:
		err = order.Deliver()

	default:
		return fmt.Errorf(
			"invalid order status %q: %w",
			newStatus,
			domain.ErrInvalidInput,
		)
	}

	// If the domain rejects the transition, return a real error
	// so order_handler.go can use:
	//
	// if err := h.orderService.UpdateStatus(...); err != nil
	//
	if err != nil {
		return fmt.Errorf("status transition failed: %w", err)
	}

	// Persist full order state after successful transition.
	// This is the key part so it aligns correctly with order_handler.go.
	if err := s.orderRepo.UpdateStatus(ctx, id, order.Status); err != nil {
		return fmt.Errorf(
			"persisting status update for order %s: %w",
			id,
			err,
		)
	}

	// Cache invalidation should never break the main flow.
	if s.cache != nil {
		_ = s.cache.Delete(ctx, fmt.Sprintf("order:%s", id))
	}

	return nil
}

// --- Internal helpers ---

// toOrderResponse converts a persisted domain.Order to an OrderResponse.
// Used by GET operations — re-fetches product names for line item enrichment.
// toOrderResponse converts a persisted domain.Order to an OrderResponse.
// Used by GET operations — re-fetches product names for line item enrichment.
func (s *orderService) toOrderResponse(ctx context.Context, order *domain.Order) (*OrderResponse, error) {
	// 1. Parsear el ID de la orden
	orderUUID, err := uuid.Parse(order.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing order ID %s: %w", order.ID, err)
	}

	// 2. Parsear el ID del usuario
	userUUID, err := uuid.Parse(order.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing user ID %s: %w", order.UserID, err)
	}

	// 3. Buscar al usuario usando el UUID ya parseado
	user, err := s.userRepo.GetByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("resolving timezone for order %s: %w", order.ID, err)
	}

	// 4. Mapear los items
	itemResponses := make([]OrderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		itemResponses = append(itemResponses, OrderItemResponse{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			ProductSKU:  item.ProductSKU,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.Subtotal,
		})
	}

	// 5. Retornar la respuesta final
	return &OrderResponse{
		ID:                    orderUUID,
		OrderNumber:           order.OrderNumber,
		UserID:                userUUID,
		Currency:              order.Currency,
		ExchangeRate:          order.ExchangeRate,
		Items:                 itemResponses,
		Subtotal:              order.Subtotal,
		TaxAmount:             order.TaxAmount,
		ShippingCost:          order.ShippingCost,
		TotalAmount:           order.TotalAmount,
		Status:                order.Status,
		CreatedAt:             order.CreatedAt,
		CreatedAtUserTimezone: convertToTimezone(order.CreatedAt, user.PreferredTimezone),
	}, nil
}

// buildOrderResponse constructs the immediate Create response from already-resolved
// product data — avoids redundant DB round-trips right after order creation.
func buildOrderResponse(order *domain.Order, items []resolvedItem, userTimezone string) *OrderResponse {
	orderUUID, _ := uuid.Parse(order.ID)
	userUUID, _ := uuid.Parse(order.UserID)

	itemResponses := make([]OrderItemResponse, 0, len(items))
	for _, ri := range items {
		itemResponses = append(itemResponses, OrderItemResponse{
			ProductID:   ri.product.ID,
			ProductName: ri.product.Name,
			ProductSKU:  ri.product.SKU,
			Quantity:    ri.quantity,
			UnitPrice:   ri.unitPrice,
			Subtotal:    ri.subtotal,
		})
	}

	return &OrderResponse{
		ID:                    orderUUID,
		OrderNumber:           order.OrderNumber,
		UserID:                userUUID,
		Currency:              order.Currency,
		ExchangeRate:          order.ExchangeRate,
		Items:                 itemResponses,
		Subtotal:              order.Subtotal,
		TaxAmount:             order.TaxAmount,
		ShippingCost:          order.ShippingCost,
		TotalAmount:           order.TotalAmount,
		Status:                order.Status,
		CreatedAt:             order.CreatedAt,
		CreatedAtUserTimezone: convertToTimezone(order.CreatedAt, userTimezone),
	}
}

// convertToTimezone formats a UTC timestamp in the user's IANA timezone.
// Falls back to UTC if the timezone string is invalid — never panics.
func convertToTimezone(t time.Time, ianaTimezone string) string {
	loc, err := time.LoadLocation(ianaTimezone)
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format(time.RFC3339)
}

// validateCreateOrderRequest enforces structural rules before any business logic runs.
func validateCreateOrderRequest(req CreateOrderRequest) error {
	if req.UserID == uuid.Nil {
		return fmt.Errorf("user_id is required: %w", domain.ErrInvalidInput)
	}
	if len(strings.TrimSpace(req.Currency)) != 3 {
		return fmt.Errorf("currency must be a valid 3-letter ISO 4217 code: %w", domain.ErrInvalidInput)
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("order must contain at least one item: %w", domain.ErrInvalidInput)
	}
	for i, item := range req.Items {
		if item.ProductID == uuid.Nil {
			return fmt.Errorf("item[%d]: product_id is required: %w", i, domain.ErrInvalidInput)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("item[%d]: quantity must be greater than zero: %w", i, domain.ErrInvalidInput)
		}
	}
	return nil
}

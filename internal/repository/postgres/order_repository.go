package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
)

type orderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) repository.OrderRepository {
	return &orderRepository{db: db}
}

// Create persists an order and all its line items atomically.
func (r *orderRepository) Create(ctx context.Context, order *domain.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction for order creation: %w", err)
	}
	defer tx.Rollback(ctx)

	orderQuery := `
		INSERT INTO orders (
			id, order_number, user_id, currency_id,
			exchange_rate, subtotal, total_amount,
			status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	_, err = tx.Exec(ctx, orderQuery,
		order.ID,
		order.OrderNumber,
		order.UserID,
		order.Currency,
		order.ExchangeRate,
		order.Subtotal,
		order.TotalAmount,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		// Aquí usamos mapPgError que ya reside en errors.go de tu proyecto
		return fmt.Errorf("inserting order header: %w", mapPgError(err))
	}

	itemQuery := `
		INSERT INTO order_items (
			id, order_id, product_id,
			quantity, unit_price, total_price,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	stockQuery := `
		UPDATE products
		SET
			stock_quantity = stock_quantity - $2,
			updated_at     = NOW()
		WHERE id = $1
		  AND is_active = true
		  AND (stock_quantity - $2) >= 0`

	for _, item := range order.Items {
		_, err = tx.Exec(ctx, itemQuery,
			item.ID,
			order.ID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
			item.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("inserting order item for product %s: %w", item.ProductID, mapPgError(err))
		}

		tag, err := tx.Exec(ctx, stockQuery, item.ProductID, item.Quantity)
		if err != nil {
			return fmt.Errorf("decrementing stock for product %s: %w", item.ProductID, err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("insufficient stock for product %s: %w", item.ProductID, domain.ErrInsufficientStock)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing order transaction: %w", err)
	}

	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	orderQuery := `
		SELECT
			id, order_number, user_id, currency_id,
			exchange_rate, subtotal, total_amount,
			status, created_at, updated_at
		FROM orders
		WHERE id = $1`

	order := &domain.Order{}
	err := r.db.QueryRow(ctx, orderQuery, id).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.UserID,
		&order.Currency,
		&order.ExchangeRate,
		&order.Subtotal,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting order by id %s: %w", id, err)
	}

	items, err := r.fetchOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

func (r *orderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
	query := `
		SELECT
			id, order_number, user_id, currency_id,
			exchange_rate, subtotal, total_amount,
			status, created_at, updated_at
		FROM orders
		WHERE order_number = $1`

	order := &domain.Order{}
	err := r.db.QueryRow(ctx, query, orderNumber).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.UserID,
		&order.Currency,
		&order.ExchangeRate,
		&order.Subtotal,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting order by number %s: %w", orderNumber, err)
	}

	items, err := r.fetchOrderItems(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

func (r *orderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, params repository.ListParams) ([]*domain.Order, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := (params.Page - 1) * params.PageSize

	var total int64
	countQuery := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting orders for user %s: %w", userID, err)
	}

	dataQuery := `
		SELECT
			id, order_number, user_id, currency_id,
			exchange_rate, subtotal, total_amount,
			status, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, dataQuery, userID, params.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing orders for user %s: %w", userID, err)
	}
	defer rows.Close()

	orders := make([]*domain.Order, 0, params.PageSize)
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.UserID,
			&order.Currency,
			&order.ExchangeRate,
			&order.Subtotal,
			&order.TotalAmount,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning order row: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, total, rows.Err()
}

func (r *orderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	query := `UPDATE orders SET status = $2, updated_at = NOW() WHERE id = $1`
	tag, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("updating status for order %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *orderRepository) List(ctx context.Context, params repository.OrderListParams) ([]*domain.Order, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := (params.Page - 1) * params.PageSize

	conditions := []string{"1 = 1"}
	args := []interface{}{}
	argIdx := 1

	if params.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, *params.UserID)
		argIdx++
	}
	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}
	if params.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *params.FromDate)
		argIdx++
	}
	if params.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *params.ToDate)
		argIdx++
	}

	whereClause := "WHERE " + joinConditions(conditions)

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders %s", whereClause)
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting orders: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT
			id, order_number, user_id, currency_id,
			exchange_rate, subtotal, total_amount,
			status, created_at, updated_at
		FROM orders
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	dataArgs := append(args, params.PageSize, offset)
	rows, err := r.db.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing orders: %w", err)
	}
	defer rows.Close()

	orders := make([]*domain.Order, 0, params.PageSize)
	for rows.Next() {
		order := &domain.Order{}
		if err := rows.Scan(
			&order.ID,
			&order.OrderNumber,
			&order.UserID,
			&order.Currency,
			&order.ExchangeRate,
			&order.Subtotal,
			&order.TotalAmount,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning order row: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, total, rows.Err()
}

func (r *orderRepository) fetchOrderItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, quantity, unit_price, total_price, created_at
		FROM order_items
		WHERE order_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("fetching items for order %s: %w", orderID, err)
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		item := domain.OrderItem{}
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning order item row: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func joinConditions(conditions []string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}

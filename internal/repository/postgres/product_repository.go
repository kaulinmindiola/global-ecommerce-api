package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/repository"
)

// productRepository implements repository.ProductRepository using PostgreSQL.
type productRepository struct {
	db *pgxpool.Pool
}

// NewProductRepository creates a new PostgreSQL-backed ProductRepository.
func NewProductRepository(db *pgxpool.Pool) repository.ProductRepository {
	return &productRepository{db: db}
}

// Create inserts a new product. The SKU must be unique across the catalogue.
func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	query := `
		INSERT INTO products (
			id, name, description, sku,
			base_price, base_currency_id,
			stock_quantity, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	_, err := r.db.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.SKU,
		product.BasePrice,
		product.BaseCurrencyID,
		product.StockQuantity,
		product.IsActive,
		product.CreatedAt,
		product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating product: %w", mapPgError(err))
	}

	return nil
}

// GetByID fetches a product by UUID, including its base currency details via JOIN.
func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	query := `
		SELECT
			p.id, p.name, p.description, p.sku,
			p.base_price, p.base_currency_id,
			p.stock_quantity, p.is_active,
			p.created_at, p.updated_at,
			c.code AS currency_code
		FROM products p
		JOIN currencies c ON c.id = p.base_currency_id
		WHERE p.id = $1 AND p.is_active = true`

	product := &domain.Product{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.SKU,
		&product.BasePrice,
		&product.BaseCurrencyID,
		&product.StockQuantity,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.BaseCurrencyCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting product by id %s: %w", id, err)
	}

	return product, nil
}

// GetBySKU fetches a product by its Stock Keeping Unit identifier.
func (r *productRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	query := `
		SELECT
			p.id, p.name, p.description, p.sku,
			p.base_price, p.base_currency_id,
			p.stock_quantity, p.is_active,
			p.created_at, p.updated_at,
			c.code AS currency_code
		FROM products p
		JOIN currencies c ON c.id = p.base_currency_id
		WHERE p.sku = $1 AND p.is_active = true`

	product := &domain.Product{}
	err := r.db.QueryRow(ctx, query, sku).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.SKU,
		&product.BasePrice,
		&product.BaseCurrencyID,
		&product.StockQuantity,
		&product.IsActive,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.BaseCurrencyCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting product by sku %s: %w", sku, err)
	}

	return product, nil
}

// Update persists mutable changes to an existing product.
func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	query := `
		UPDATE products SET
			name             = $2,
			description      = $3,
			base_price       = $4,
			base_currency_id = $5,
			updated_at       = $6
		WHERE id = $1 AND is_active = true`

	tag, err := r.db.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.BasePrice,
		product.BaseCurrencyID,
		product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating product %s: %w", product.ID, mapPgError(err))
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete soft-deletes a product by setting is_active = false.
func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE products SET is_active = false, updated_at = NOW()
		WHERE id = $1 AND is_active = true`

	tag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting product %s: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// List retrieves a paginated, filtered list of active products.
// Filters are applied dynamically — only non-zero values are included in the WHERE clause.
func (r *productRepository) List(ctx context.Context, params repository.ProductListParams) ([]*domain.Product, int64, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}
	offset := (params.Page - 1) * params.PageSize

	// Build dynamic WHERE clauses using a slice of conditions and args.
	conditions := []string{"p.is_active = true"}
	args := []interface{}{}
	argIdx := 1

	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(p.name ILIKE $%d OR p.sku ILIKE $%d)", argIdx, argIdx+1,
		))
		searchPattern := "%" + params.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIdx += 2
	}

	if params.CurrencyCode != "" {
		conditions = append(conditions, fmt.Sprintf("c.code = $%d", argIdx))
		args = append(args, params.CurrencyCode)
		argIdx++
	}

	if params.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("p.base_price >= $%d", argIdx))
		args = append(args, *params.MinPrice)
		argIdx++
	}

	if params.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("p.base_price <= $%d", argIdx))
		args = append(args, *params.MaxPrice)
		argIdx++
	}

	if params.InStockOnly {
		conditions = append(conditions, "p.stock_quantity > 0")
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total matching records.
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM products p
		JOIN currencies c ON c.id = p.base_currency_id
		%s`, whereClause)

	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting products: %w", err)
	}

	// Fetch paginated data.
	dataArgs := append(args, params.PageSize, offset)
	dataQuery := fmt.Sprintf(`
		SELECT
			p.id, p.name, p.description, p.sku,
			p.base_price, p.base_currency_id,
			p.stock_quantity, p.is_active,
			p.created_at, p.updated_at,
			c.code AS currency_code
		FROM products p
		JOIN currencies c ON c.id = p.base_currency_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	rows, err := r.db.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing products: %w", err)
	}
	defer rows.Close()

	products := make([]*domain.Product, 0, params.PageSize)
	for rows.Next() {
		product := &domain.Product{}
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.SKU,
			&product.BasePrice,
			&product.BaseCurrencyID,
			&product.StockQuantity,
			&product.IsActive,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.BaseCurrencyCode,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning product row: %w", err)
		}
		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating product rows: %w", err)
	}

	return products, total, nil
}

// UpdateStock atomically adjusts stock using a database-level check constraint.
// Using UPDATE with a conditional prevents race conditions under concurrent requests.
func (r *productRepository) UpdateStock(ctx context.Context, id uuid.UUID, delta int) error {
	query := `
		UPDATE products
		SET
			stock_quantity = stock_quantity + $2,
			updated_at     = NOW()
		WHERE id = $1
		  AND is_active = true
		  AND (stock_quantity + $2) >= 0`

	tag, err := r.db.Exec(ctx, query, id, delta)
	if err != nil {
		return fmt.Errorf("updating stock for product %s: %w", id, err)
	}

	// RowsAffected = 0 means either product not found OR stock would go negative.
	if tag.RowsAffected() == 0 {
		// Distinguish between not-found and insufficient stock.
		var exists bool
		checkQuery := `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1 AND is_active = true)`
		if err := r.db.QueryRow(ctx, checkQuery, id).Scan(&exists); err != nil {
			return fmt.Errorf("checking product existence %s: %w", id, err)
		}
		if !exists {
			return domain.ErrNotFound
		}
		return domain.ErrInsufficientStock
	}

	return nil
}

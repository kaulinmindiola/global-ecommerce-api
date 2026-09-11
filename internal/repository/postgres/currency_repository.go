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

// currencyRepository implements repository.CurrencyRepository using PostgreSQL.
type currencyRepository struct {
	db *pgxpool.Pool
}

// NewCurrencyRepository creates a new PostgreSQL-backed CurrencyRepository.
func NewCurrencyRepository(db *pgxpool.Pool) repository.CurrencyRepository {
	return &currencyRepository{db: db}
}

// Create inserts a new supported currency into the system.
// The ISO code (e.g., "USD", "EUR") must be unique.
func (r *currencyRepository) Create(ctx context.Context, currency *domain.Currency) error {
	query := `
		INSERT INTO currencies (
			id, code, name, symbol,
			exchange_rate_to_usd, is_active,
			rate_updated_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.Exec(ctx, query,
		currency.ID,
		currency.Code,
		currency.Name,
		currency.Symbol,
		currency.ExchangeRateToUSD,
		currency.IsActive,
		currency.RateUpdatedAt,
		currency.CreatedAt,
		currency.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("creating currency: %w", mapPgError(err))
	}

	return nil
}

// GetByID fetches a currency by its UUID.
func (r *currencyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Currency, error) {
	query := `
		SELECT
			id, code, name, symbol,
			exchange_rate_to_usd, is_active,
			rate_updated_at, created_at, updated_at
		FROM currencies
		WHERE id = $1`

	currency := &domain.Currency{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&currency.ID,
		&currency.Code,
		&currency.Name,
		&currency.Symbol,
		&currency.ExchangeRateToUSD,
		&currency.IsActive,
		&currency.RateUpdatedAt,
		&currency.CreatedAt,
		&currency.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting currency by id %s: %w", id, err)
	}

	return currency, nil
}

// GetByCode fetches a currency by its ISO 4217 alphabetic code.
// The lookup is case-insensitive (UPPER normalization applied at insertion).
func (r *currencyRepository) GetByCode(ctx context.Context, code string) (*domain.Currency, error) {
	query := `
		SELECT
			id, code, name, symbol,
			exchange_rate_to_usd, is_active,
			rate_updated_at, created_at, updated_at
		FROM currencies
		WHERE code = UPPER($1)`

	currency := &domain.Currency{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&currency.ID,
		&currency.Code,
		&currency.Name,
		&currency.Symbol,
		&currency.ExchangeRateToUSD,
		&currency.IsActive,
		&currency.RateUpdatedAt,
		&currency.CreatedAt,
		&currency.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("getting currency by code %s: %w", code, err)
	}

	return currency, nil
}

// ListActive returns all currencies with is_active = true,
// ordered alphabetically by ISO code for consistent API responses.
func (r *currencyRepository) ListActive(ctx context.Context) ([]*domain.Currency, error) {
	query := `
		SELECT
			id, code, name, symbol,
			exchange_rate_to_usd, is_active,
			rate_updated_at, created_at, updated_at
		FROM currencies
		WHERE is_active = true
		ORDER BY code ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing active currencies: %w", err)
	}
	defer rows.Close()

	currencies := make([]*domain.Currency, 0)
	for rows.Next() {
		currency := &domain.Currency{}
		if err := rows.Scan(
			&currency.ID,
			&currency.Code,
			&currency.Name,
			&currency.Symbol,
			&currency.ExchangeRateToUSD,
			&currency.IsActive,
			&currency.RateUpdatedAt,
			&currency.CreatedAt,
			&currency.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning currency row: %w", err)
		}
		currencies = append(currencies, currency)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating currency rows: %w", err)
	}

	return currencies, nil
}

// Update persists changes to a currency's name, symbol, and active status.
// The ISO code is immutable once created.
func (r *currencyRepository) Update(ctx context.Context, currency *domain.Currency) error {
	query := `
		UPDATE currencies SET
			name       = $2,
			symbol     = $3,
			is_active  = $4,
			updated_at = $5
		WHERE id = $1`

	tag, err := r.db.Exec(ctx, query,
		currency.ID,
		currency.Name,
		currency.Symbol,
		currency.IsActive,
		currency.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("updating currency %s: %w", currency.ID, mapPgError(err))
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateExchangeRate updates the exchange rate relative to USD for a given currency.
// rate_updated_at is set to NOW() at the database level to ensure UTC precision.
//
// Financial note: exchange rates are stored as NUMERIC in the database to avoid
// floating-point rounding errors that would be unacceptable in monetary calculations.
func (r *currencyRepository) UpdateExchangeRate(ctx context.Context, id uuid.UUID, rate float64) error {
	query := `
		UPDATE currencies SET
			exchange_rate_to_usd = $2,
			rate_updated_at      = NOW(),
			updated_at           = NOW()
		WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id, rate)
	if err != nil {
		return fmt.Errorf("updating exchange rate for currency %s: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

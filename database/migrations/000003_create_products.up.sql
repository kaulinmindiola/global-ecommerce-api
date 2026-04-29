-- =============================================================================
-- Migration: 000003_create_products
-- Description: Create the products table for the global product catalogue.
--
-- Design decisions:
--   • base_price uses NUMERIC(19,4) — 4 decimal places matches the service
--     layer's .Round(4) calls. FLOAT is never used for monetary values.
--   • base_currency_id links each product to its native pricing currency.
--     The service layer converts to any other currency at request time.
--   • sku (Stock Keeping Unit) is a business identifier — unique and uppercase.
--   • stock_quantity uses CHECK >= 0 at DB level as a last-resort guard.
--     The primary stock validation happens in the service layer (atomic UPDATE).
--   • Soft delete: is_active = false — orders can still reference deleted products.
-- =============================================================================

CREATE TABLE IF NOT EXISTS products (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(255)  NOT NULL,
    description       TEXT,
    sku               VARCHAR(100)  NOT NULL,
    base_price        NUMERIC(19,4) NOT NULL,
    base_currency_id  UUID          NOT NULL REFERENCES currencies(id) ON DELETE RESTRICT,
    stock_quantity    INTEGER       NOT NULL DEFAULT 0,
    is_active         BOOLEAN       NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Price must be positive — a zero-price product is a data error.
    CONSTRAINT products_price_positive CHECK (base_price > 0),

    -- Stock cannot go negative — enforced here AND in the service layer's atomic UPDATE.
    CONSTRAINT products_stock_non_negative CHECK (stock_quantity >= 0),

    -- SKU must be uppercase for consistent lookups.
    CONSTRAINT products_sku_uppercase CHECK (sku = UPPER(sku)),

    -- SKU must be unique across the catalogue.
    CONSTRAINT products_sku_unique UNIQUE (sku)
);

-- Primary lookup: get product by SKU.
CREATE UNIQUE INDEX IF NOT EXISTS idx_products_sku
    ON products (sku)
    WHERE is_active = true;

-- Catalogue listing: active products sorted by creation date.
CREATE INDEX IF NOT EXISTS idx_products_active
    ON products (is_active, created_at DESC);

-- Currency join: resolving product prices in the catalogue endpoint.
CREATE INDEX IF NOT EXISTS idx_products_currency
    ON products (base_currency_id);

-- Full-text search on product name for the search query parameter.
CREATE INDEX IF NOT EXISTS idx_products_name_search
    ON products USING GIN (to_tsvector('english', name));

-- Price range filtering used by the catalogue endpoint.
CREATE INDEX IF NOT EXISTS idx_products_price
    ON products (base_price)
    WHERE is_active = true;

-- In-stock filter: common query pattern in the catalogue endpoint.
CREATE INDEX IF NOT EXISTS idx_products_in_stock
    ON products (stock_quantity)
    WHERE is_active = true AND stock_quantity > 0;

COMMENT ON TABLE products IS
    'Product catalogue with base pricing in native currency. Prices are converted at request time.';
COMMENT ON COLUMN products.base_price IS
    'Price in base_currency. NUMERIC(19,4) avoids floating-point rounding errors in financial math.';
COMMENT ON COLUMN products.sku IS
    'Stock Keeping Unit — unique business identifier, always uppercase.';

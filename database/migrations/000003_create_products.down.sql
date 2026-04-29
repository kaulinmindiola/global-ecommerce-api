-- =============================================================================
-- Rollback: 000003_create_products
-- =============================================================================

DROP INDEX IF EXISTS idx_products_in_stock;
DROP INDEX IF EXISTS idx_products_price;
DROP INDEX IF EXISTS idx_products_name_search;
DROP INDEX IF EXISTS idx_products_currency;
DROP INDEX IF EXISTS idx_products_active;
DROP INDEX IF EXISTS idx_products_sku;
DROP TABLE IF EXISTS products;

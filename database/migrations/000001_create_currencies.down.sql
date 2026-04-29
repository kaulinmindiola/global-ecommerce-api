-- =============================================================================
-- Rollback: 000001_create_currencies
-- Drops all objects created by the up migration in reverse dependency order.
-- =============================================================================

DROP INDEX IF EXISTS idx_currencies_active;
DROP INDEX IF EXISTS idx_currencies_code;
DROP TABLE IF EXISTS currencies;

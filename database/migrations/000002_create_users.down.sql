-- =============================================================================
-- Rollback: 000002_create_users
-- =============================================================================

DROP INDEX IF EXISTS idx_users_currency;
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;

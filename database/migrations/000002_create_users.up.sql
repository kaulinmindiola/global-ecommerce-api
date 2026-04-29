-- =============================================================================
-- Migration: 000002_create_users
-- Description: Create the users table for global customer management.
--
-- Design decisions:
--   • password_hash stores bcrypt output only — raw passwords NEVER stored.
--   • preferred_currency_id is a FK to currencies — validated at DB level.
--   • preferred_timezone stores IANA timezone strings (e.g. "America/Bogota").
--     Validated at the application layer via time.LoadLocation.
--   • Soft delete: is_active = false preserves audit history instead of
--     physically deleting rows. Required for financial compliance.
--   • email is stored lowercase (enforced by CHECK) — uniqueness is
--     case-insensitive in practice.
-- =============================================================================

CREATE TABLE IF NOT EXISTS users (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 VARCHAR(255) NOT NULL,
    full_name             VARCHAR(255) NOT NULL,
    password_hash         VARCHAR(255) NOT NULL,
    preferred_currency_id UUID         NOT NULL REFERENCES currencies(id) ON DELETE RESTRICT,
    preferred_timezone    VARCHAR(100) NOT NULL DEFAULT 'UTC',
    is_active             BOOLEAN      NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Email must be lowercase for case-insensitive uniqueness.
    CONSTRAINT users_email_lowercase CHECK (email = LOWER(email)),

    -- Email format sanity check (full validation is done at the application layer).
    CONSTRAINT users_email_format CHECK (email ~* '^[^@]+@[^@]+\.[^@]+$'),

    -- Full name must have at least 2 characters.
    CONSTRAINT users_full_name_length CHECK (LENGTH(TRIM(full_name)) >= 2),

    -- Unique email across active and inactive users — prevents re-registration.
    CONSTRAINT users_email_unique UNIQUE (email)
);

-- Primary lookup: login by email.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users (email)
    WHERE is_active = true;

-- Listing queries: filter by active status.
CREATE INDEX IF NOT EXISTS idx_users_active
    ON users (is_active, created_at DESC);

-- FK join: resolving preferred currency for user profile responses.
CREATE INDEX IF NOT EXISTS idx_users_currency
    ON users (preferred_currency_id);

COMMENT ON TABLE users IS
    'Registered customers with their preferred currency and timezone settings.';
COMMENT ON COLUMN users.password_hash IS
    'bcrypt hash of the user password. Raw password is NEVER stored.';
COMMENT ON COLUMN users.preferred_timezone IS
    'IANA timezone identifier (e.g. America/Bogota). Used to localise timestamps in responses.';
COMMENT ON COLUMN users.is_active IS
    'Soft delete flag. Set to false instead of physically deleting for audit compliance.';

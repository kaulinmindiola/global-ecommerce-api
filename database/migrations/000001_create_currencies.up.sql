-- =============================================================================
-- Migration: 000001_create_currencies
-- Description: Create the currencies table for multi-currency support.
--
-- Design decisions:
--   • exchange_rate_to_usd uses NUMERIC(18,8) — 8 decimal places for precision
--     required by financial calculations. FLOAT is never used for money.
--   • All timestamps use TIMESTAMPTZ (timezone-aware), stored in UTC.
--     This is mandatory for a global platform handling multiple timezones.
--   • code is CHAR(3) and stored uppercase — enforced by CHECK constraint.
--   • USD is the pivot currency: exchange_rate_to_usd = 1.0 for USD.
-- =============================================================================

CREATE TABLE IF NOT EXISTS currencies (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code                 CHAR(3)      NOT NULL,
    name                 VARCHAR(100) NOT NULL,
    symbol               VARCHAR(10)  NOT NULL,
    exchange_rate_to_usd NUMERIC(18,8) NOT NULL DEFAULT 1.0,
    is_active            BOOLEAN      NOT NULL DEFAULT true,
    rate_updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- ISO 4217 requires exactly 3 uppercase letters.
    CONSTRAINT currencies_code_format CHECK (code ~ '^[A-Z]{3}$'),

    -- Exchange rate must always be positive — a zero rate would break conversion.
    CONSTRAINT currencies_exchange_rate_positive CHECK (exchange_rate_to_usd > 0),

    -- Unique code ensures no duplicate currency registrations.
    CONSTRAINT currencies_code_unique UNIQUE (code)
);

-- Index for frequent lookups by ISO code (used in every currency conversion).
CREATE INDEX IF NOT EXISTS idx_currencies_code
    ON currencies (code)
    WHERE is_active = true;

-- Index to quickly retrieve all active currencies for the list endpoint.
CREATE INDEX IF NOT EXISTS idx_currencies_active
    ON currencies (is_active);

COMMENT ON TABLE currencies IS
    'Supported currencies with exchange rates relative to USD as the pivot currency.';
COMMENT ON COLUMN currencies.exchange_rate_to_usd IS
    '1 unit of this currency = N USD. USD itself has rate 1.0.';
COMMENT ON COLUMN currencies.rate_updated_at IS
    'Last time the exchange rate was refreshed. Used for stale rate detection.';

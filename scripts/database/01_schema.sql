-- ============================================================================
-- GLOBAL E-COMMERCE MICROSERVICES API - DATABASE SCHEMA
-- Author: Kaulin Mindiola
-- Purpose: Demonstrate senior-level database design for international commerce
-- Key Features: Multi-currency support, timezone awareness, audit trail
-- ============================================================================

-- Enable UUID extension for better distributed system compatibility
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLE: currencies
-- Purpose: Dynamic currency catalog supporting ISO 4217 standards
-- Business Value: Enables expansion to new markets without code changes
-- ============================================================================
CREATE TABLE currencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(3) NOT NULL UNIQUE, -- ISO 4217 (USD, EUR, COP, JPY, etc.)
    name VARCHAR(50) NOT NULL,
    symbol VARCHAR(5) NOT NULL,
    decimal_places SMALLINT NOT NULL DEFAULT 2, -- JPY has 0, most have 2
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast currency lookups during order processing
CREATE INDEX idx_currencies_code ON currencies(code) WHERE is_active = true;

-- ============================================================================
-- TABLE: users
-- Purpose: Global customer base with localization preferences
-- Senior Detail: Stores timezone and preferred currency for personalization
-- ============================================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    full_name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- bcrypt recommended
    preferred_currency_id UUID REFERENCES currencies(id),
    preferred_timezone VARCHAR(50) NOT NULL DEFAULT 'UTC', -- IANA timezone
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Performance indexes for authentication and user lookup
CREATE INDEX idx_users_email ON users(email) WHERE is_active = true;
CREATE INDEX idx_users_preferred_currency ON users(preferred_currency_id);

-- ============================================================================
-- TABLE: products
-- Purpose: Product catalog with base pricing
-- Critical Design: Uses NUMERIC for financial precision (avoid FLOAT!)
-- ============================================================================
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sku VARCHAR(100) NOT NULL UNIQUE, -- Stock Keeping Unit
    base_price NUMERIC(15, 4) NOT NULL, -- High precision for currency conversion
    base_currency_id UUID NOT NULL REFERENCES currencies(id),
    stock_quantity INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_base_price_positive CHECK (base_price > 0),
    CONSTRAINT chk_stock_non_negative CHECK (stock_quantity >= 0)
);

-- Indexes for product search and filtering
CREATE INDEX idx_products_sku ON products(sku) WHERE is_active = true;
CREATE INDEX idx_products_base_currency ON products(base_currency_id);
CREATE INDEX idx_products_active ON products(is_active);

-- ============================================================================
-- TABLE: exchange_rates
-- Purpose: Historical currency conversion rates
-- Audit Trail: Critical for financial reconciliation and compliance
-- ============================================================================
CREATE TABLE exchange_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_currency_id UUID NOT NULL REFERENCES currencies(id),
    to_currency_id UUID NOT NULL REFERENCES currencies(id),
    rate NUMERIC(18, 8) NOT NULL, -- High precision for accurate conversion
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    source VARCHAR(50), -- e.g., 'ECB', 'OpenExchangeRates', 'Manual'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_rate_positive CHECK (rate > 0),
    CONSTRAINT chk_different_currencies CHECK (from_currency_id != to_currency_id)
);

-- Composite index for fast rate lookups during checkout
CREATE INDEX idx_exchange_rates_lookup
    ON exchange_rates(from_currency_id, to_currency_id, effective_date DESC);

-- ============================================================================
-- TABLE: orders
-- Purpose: Purchase transactions with currency snapshot
-- Senior Pattern: Stores exchange_rate at time of purchase for immutability
-- Why: Exchange rates fluctuate; we need historical accuracy for refunds/reports
-- ============================================================================
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_number VARCHAR(50) NOT NULL UNIQUE, -- Human-readable: ORD-2025-001234
    user_id UUID NOT NULL REFERENCES users(id),

    -- Currency information snapshot
    currency_id UUID NOT NULL REFERENCES currencies(id),
    exchange_rate NUMERIC(18, 8) NOT NULL, -- Rate used at purchase time

    -- Pricing breakdown
    subtotal NUMERIC(15, 4) NOT NULL,
    tax_amount NUMERIC(15, 4) NOT NULL DEFAULT 0,
    shipping_cost NUMERIC(15, 4) NOT NULL DEFAULT 0,
    total_amount NUMERIC(15, 4) NOT NULL,

    -- Order lifecycle
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, confirmed, shipped, delivered, cancelled

    -- Audit timestamps (UTC for global consistency)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,

    CONSTRAINT chk_subtotal_positive CHECK (subtotal > 0),
    CONSTRAINT chk_total_positive CHECK (total_amount > 0)
);

-- Indexes for order management and reporting
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created ON orders(created_at DESC);
CREATE INDEX idx_orders_number ON orders(order_number);

-- ============================================================================
-- TABLE: order_items
-- Purpose: Line items within each order
-- Design: Stores product snapshot (name, price) for historical accuracy
-- Why: Product prices may change; orders must reflect purchase-time values
-- ============================================================================
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),

    -- Product snapshot at time of purchase
    product_name VARCHAR(255) NOT NULL,
    product_sku VARCHAR(100) NOT NULL,
    unit_price NUMERIC(15, 4) NOT NULL, -- Price in order's currency
    quantity INTEGER NOT NULL,
    subtotal NUMERIC(15, 4) NOT NULL, -- unit_price * quantity

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_unit_price_positive CHECK (unit_price > 0)
);

-- Index for retrieving items by order
CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_product ON order_items(product_id);

-- ============================================================================
-- TRIGGERS: Automatic timestamp updates
-- Professional Touch: Ensures updated_at is always current
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to relevant tables
CREATE TRIGGER set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER set_updated_at BEFORE UPDATE ON currencies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SEED DATA: Initial currencies for testing
-- ============================================================================
INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
    ('USD', 'US Dollar', '$', 2),
    ('EUR', 'Euro', '€', 2),
    ('COP', 'Colombian Peso', '$', 2),
    ('GBP', 'British Pound', '£', 2),
    ('JPY', 'Japanese Yen', '¥', 0);

-- ============================================================================
-- COMMENTS: Documentation for future developers
-- ============================================================================
COMMENT ON TABLE orders IS 'Purchase transactions with immutable currency snapshot for audit compliance';
COMMENT ON COLUMN orders.exchange_rate IS 'Rate used at purchase time - never update this value';
COMMENT ON COLUMN products.base_price IS 'Uses NUMERIC to prevent floating-point precision errors in financial calculations';
COMMENT ON TABLE exchange_rates IS 'Historical currency conversion rates - append-only for regulatory compliance';

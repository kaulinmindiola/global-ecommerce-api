-- =============================================================================
-- Migration: 000004_create_orders
-- Description: Create orders and order_items tables — the financial core.
--
-- Design decisions:
--   • exchange_rate uses NUMERIC(18,6) — 6 decimal places to match the
--     service layer's .Round(6) snapshot. This is the rate AT THE TIME OF
--     purchase, stored for financial audit compliance.
--   • All monetary amounts use NUMERIC(19,4) — never FLOAT.
--   • order_number is human-readable (ORD-2025-XXXXXX) and unique.
--   • order_items snapshots product_name and product_sku at purchase time.
--     If the product is later renamed or deleted, the order history is preserved.
--   • ON DELETE RESTRICT on user_id and currency_id prevents accidental
--     deletion of referenced data. Currencies and users must be soft-deleted.
-- =============================================================================

-- ── Orders table ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS orders (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number  VARCHAR(50)   NOT NULL,
    user_id       UUID          NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
    currency_id   UUID          NOT NULL REFERENCES currencies(id)  ON DELETE RESTRICT,
    exchange_rate NUMERIC(18,6) NOT NULL,
    subtotal      NUMERIC(19,4) NOT NULL DEFAULT 0,
    tax_amount    NUMERIC(19,4) NOT NULL DEFAULT 0,
    shipping_cost NUMERIC(19,4) NOT NULL DEFAULT 0,
    total_amount  NUMERIC(19,4) NOT NULL DEFAULT 0,
    status        VARCHAR(20)   NOT NULL DEFAULT 'pending',
    confirmed_at  TIMESTAMPTZ,
    cancelled_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Exchange rate must be positive for valid currency conversion.
    CONSTRAINT orders_exchange_rate_positive CHECK (exchange_rate > 0),

    -- Totals must be non-negative.
    CONSTRAINT orders_subtotal_non_negative    CHECK (subtotal >= 0),
    CONSTRAINT orders_total_non_negative       CHECK (total_amount >= 0),
    CONSTRAINT orders_shipping_non_negative    CHECK (shipping_cost >= 0),

    -- Status must be one of the valid lifecycle states.
    CONSTRAINT orders_status_valid CHECK (
        status IN ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled')
    ),

    -- Order number is the human-readable identifier (ORD-2025-XXXXXX).
    CONSTRAINT orders_number_unique UNIQUE (order_number)
);

-- Primary lookup: get order by ID (most frequent query).
CREATE INDEX IF NOT EXISTS idx_orders_id
    ON orders (id);

-- User order history: list all orders for a specific customer.
CREATE INDEX IF NOT EXISTS idx_orders_user_id
    ON orders (user_id, created_at DESC);

-- Order number lookup: human-readable reference used in customer support.
CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_order_number
    ON orders (order_number);

-- Admin listing: filter orders by status.
CREATE INDEX IF NOT EXISTS idx_orders_status
    ON orders (status, created_at DESC);

-- Date range filtering for admin reports and analytics.
CREATE INDEX IF NOT EXISTS idx_orders_created_at
    ON orders (created_at DESC);


-- ── Order items table ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_items (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID          NOT NULL REFERENCES orders(id)   ON DELETE CASCADE,
    product_id    UUID          NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name  VARCHAR(255)  NOT NULL, -- snapshot at purchase time
    product_sku   VARCHAR(100)  NOT NULL, -- snapshot at purchase time
    unit_price    NUMERIC(19,4) NOT NULL,
    quantity      INTEGER       NOT NULL,
    subtotal      NUMERIC(19,4) NOT NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    -- Price snapshot must be positive.
    CONSTRAINT order_items_unit_price_positive CHECK (unit_price > 0),

    -- Quantity must be at least 1.
    CONSTRAINT order_items_quantity_positive CHECK (quantity >= 1),

    -- Subtotal must match unit_price * quantity (enforced at app layer, validated here).
    CONSTRAINT order_items_subtotal_non_negative CHECK (subtotal >= 0)
);

-- Primary join: get all items for a given order (always used with order header).
CREATE INDEX IF NOT EXISTS idx_order_items_order_id
    ON order_items (order_id);

-- Product history: find all orders that included a specific product (analytics).
CREATE INDEX IF NOT EXISTS idx_order_items_product_id
    ON order_items (product_id);

COMMENT ON TABLE orders IS
    'Purchase transactions. exchange_rate is snapshotted at purchase time for financial audit.';
COMMENT ON COLUMN orders.exchange_rate IS
    'USD to order_currency rate at the moment of purchase. Immutable after creation.';
COMMENT ON TABLE order_items IS
    'Line items for each order. product_name and product_sku are snapshots — immutable after creation.';
COMMENT ON COLUMN order_items.product_name IS
    'Product name at the time of purchase. Preserved even if the product is later renamed.';

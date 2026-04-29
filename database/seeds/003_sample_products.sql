-- =============================================================================
-- Seed: 003_sample_products
-- Description: Insert sample products for development and demo purposes.
--
-- Products span multiple base currencies to demonstrate multi-currency
-- price conversion in the catalogue endpoint.
-- =============================================================================

INSERT INTO products (
    id,
    name,
    description,
    sku,
    base_price,
    base_currency_id,
    stock_quantity,
    is_active,
    created_at,
    updated_at
)
VALUES
    -- Electronics (USD)
    (
        '20000000-0000-4000-a000-000000000001',
        'Wireless Bluetooth Headphones',
        'Premium noise-canceling headphones with 30-hour battery life and active noise cancellation.',
        'AUDIO-WH-001',
        149.99,
        'c1000000-0000-4000-a000-000000000001', -- USD
        45,
        true,
        NOW(), NOW()
    ),
    (
        '20000000-0000-4000-a000-000000000002',
        '4K Ultra HD Monitor 27"',
        '27-inch professional display with 144Hz refresh rate, HDR400, and USB-C connectivity.',
        'DISPLAY-4K-27',
        399.99,
        'c1000000-0000-4000-a000-000000000001', -- USD
        30,
        true,
        NOW(), NOW()
    ),
    (
        '20000000-0000-4000-a000-000000000003',
        'Mechanical Keyboard TKL',
        'Tenkeyless mechanical keyboard with Cherry MX Red switches and RGB backlighting.',
        'KB-MEC-TKL-001',
        89.99,
        'c1000000-0000-4000-a000-000000000001', -- USD
        60,
        true,
        NOW(), NOW()
    ),
    (
        '20000000-0000-4000-a000-000000000004',
        'USB-C Hub 7-in-1',
        'Multiport adapter with 4K HDMI, 100W PD charging, 2x USB-A 3.0, SD/MicroSD card readers.',
        'HUB-USBC-7IN1',
        49.99,
        'c1000000-0000-4000-a000-000000000001', -- USD
        120,
        true,
        NOW(), NOW()
    ),

    -- Accessories (EUR — demonstrates cross-currency conversion)
    (
        '20000000-0000-4000-a000-000000000005',
        'Laptop Backpack 15.6"',
        'Water-resistant backpack with dedicated laptop compartment, USB charging port, and TSA-friendly design.',
        'BAG-LAPTOP-156',
        79.99,
        'c1000000-0000-4000-a000-000000000002', -- EUR
        80,
        true,
        NOW(), NOW()
    ),
    (
        '20000000-0000-4000-a000-000000000006',
        'Ergonomic Office Chair',
        'Lumbar support office chair with adjustable armrests, height, and breathable mesh back.',
        'CHAIR-ERGO-001',
        299.99,
        'c1000000-0000-4000-a000-000000000002', -- EUR
        15,
        true,
        NOW(), NOW()
    ),

    -- Books / Software (GBP — further demonstrates multi-currency)
    (
        '20000000-0000-4000-a000-000000000007',
        'Clean Architecture: A Craftsman Guide',
        'Robert C. Martin''s guide to writing clean, maintainable software architecture. Hardcover.',
        'BOOK-CLEAN-ARCH',
        34.99,
        'c1000000-0000-4000-a000-000000000003', -- GBP
        200,
        true,
        NOW(), NOW()
    ),
    (
        '20000000-0000-4000-a000-000000000008',
        'Designing Data-Intensive Applications',
        'Martin Kleppmann''s comprehensive guide to building reliable, scalable, and maintainable systems.',
        'BOOK-DDIA-001',
        44.99,
        'c1000000-0000-4000-a000-000000000003', -- GBP
        150,
        true,
        NOW(), NOW()
    ),

    -- Out of stock product (for testing edge cases)
    (
        '20000000-0000-4000-a000-000000000009',
        'Standing Desk Converter',
        'Adjustable desk riser that converts any desk to a standing desk. Capacity: 15kg.',
        'DESK-CONV-001',
        189.99,
        'c1000000-0000-4000-a000-000000000001', -- USD
        0,                                       -- out of stock
        true,
        NOW(), NOW()
    ),

    -- Inactive product (for testing soft-delete)
    (
        '20000000-0000-4000-a000-000000000010',
        'Legacy USB 2.0 Hub',
        'Discontinued 4-port USB 2.0 hub. Replaced by USB-C Hub 7-in-1.',
        'HUB-USB2-4P',
        9.99,
        'c1000000-0000-4000-a000-000000000001', -- USD
        0,
        false,                                   -- soft-deleted
        NOW(), NOW()
    )

ON CONFLICT (sku) DO NOTHING;

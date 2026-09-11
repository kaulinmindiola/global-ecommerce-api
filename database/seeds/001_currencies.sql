-- =============================================================================
-- Seed: 001_currencies
-- Description: Insert supported currencies with approximate exchange rates.
--
-- Notes:
--   • USD is the pivot currency — exchange_rate_to_usd = 1.0.
--   • Rates are approximate and for development/demo purposes only.
--   • In production, rates are updated by a scheduled job via the exchange
--     rate API integration.
--   • ON CONFLICT DO NOTHING ensures this seed is idempotent — safe to re-run.
-- =============================================================================

INSERT INTO currencies (id, code, name, symbol, exchange_rate_to_usd, is_active, rate_updated_at)
VALUES
    -- Pivot currency
    ('c1000000-0000-4000-a000-000000000001', 'USD', 'US Dollar',        '$',  1.00000000, true, NOW()),

    -- Major currencies
    ('c1000000-0000-4000-a000-000000000002', 'EUR', 'Euro',             '€',  0.92500000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000003', 'GBP', 'British Pound',    '£',  0.79000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000004', 'JPY', 'Japanese Yen',     '¥', 149.5000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000005', 'CAD', 'Canadian Dollar',  '$',  1.36000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000006', 'AUD', 'Australian Dollar','$',  1.53000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000007', 'CHF', 'Swiss Franc',      'Fr', 0.90000000, true, NOW()),

    -- Latin American currencies (relevant for the project's context)
    ('c1000000-0000-4000-a000-000000000008', 'COP', 'Colombian Peso',   '$', 3950.00000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000009', 'MXN', 'Mexican Peso',     '$',  17.2000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000010', 'BRL', 'Brazilian Real',   'R$',  4.97000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000011', 'ARS', 'Argentine Peso',   '$', 882.00000000, true, NOW()),

    -- Other major markets
    ('c1000000-0000-4000-a000-000000000012', 'CNY', 'Chinese Yuan',     '¥',  7.23000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000013', 'INR', 'Indian Rupee',     '₹',  83.1000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000014', 'KRW', 'South Korean Won', '₩', 1325.0000000, true, NOW()),
    ('c1000000-0000-4000-a000-000000000015', 'SGD', 'Singapore Dollar', '$',  1.34000000, true, NOW())

ON CONFLICT (code) DO NOTHING;

-- =============================================================================
-- Seed: 002_test_users
-- Description: Insert test users for development and integration testing.
--
-- Passwords (bcrypt cost 10, safe to commit since these are dev-only):
--   admin@example.com   → SecurePass123!
--   kaulin@example.com  → SecurePass123!
--   test@example.com    → SecurePass123!
--
-- ⚠ NEVER run this seed in production. CI/CD pipeline skips seeds in prod.
-- =============================================================================

INSERT INTO users (
    id,
    email,
    full_name,
    password_hash,
    preferred_currency_id,
    preferred_timezone,
    is_active,
    created_at,
    updated_at
)
VALUES
    (
        '10000000-0000-4000-a000-000000000001',
        'admin@example.com',
        'Admin User',
        -- bcrypt hash of 'SecurePass123!' with cost=10
        '$2a$10$rQnm2.y4xK3vJ5bL6dF8eOH1mN7pS9cT0wU3vX4yZ5aB6cD7eF8gH',
        'c1000000-0000-4000-a000-000000000001', -- USD
        'UTC',
        true,
        NOW(),
        NOW()
    ),
    (
        '10000000-0000-4000-a000-000000000002',
        'kaulin@example.com',
        'Kaulin Mindiola',
        -- bcrypt hash of 'SecurePass123!' with cost=10
        '$2a$10$rQnm2.y4xK3vJ5bL6dF8eOH1mN7pS9cT0wU3vX4yZ5aB6cD7eF8gH',
        'c1000000-0000-4000-a000-000000000008', -- COP
        'America/Bogota',
        true,
        NOW(),
        NOW()
    ),
    (
        '10000000-0000-4000-a000-000000000003',
        'test.europe@example.com',
        'European Test User',
        -- bcrypt hash of 'SecurePass123!' with cost=10
        '$2a$10$rQnm2.y4xK3vJ5bL6dF8eOH1mN7pS9cT0wU3vX4yZ5aB6cD7eF8gH',
        'c1000000-0000-4000-a000-000000000002', -- EUR
        'Europe/Madrid',
        true,
        NOW(),
        NOW()
    )

ON CONFLICT (email) DO NOTHING;

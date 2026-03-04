-- Insert sample data for testing
-- Sample users (password is 'password123' hashed with bcrypt)
INSERT INTO users (id, email, password_hash, role) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'admin@marketplace.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'ADMIN'),
    ('550e8400-e29b-41d4-a716-446655440002', 'seller@marketplace.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'SELLER'),
    ('550e8400-e29b-41d4-a716-446655440003', 'user@marketplace.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'USER')
ON CONFLICT (email) DO NOTHING;

-- Sample products
INSERT INTO products (id, name, description, price, stock, category, status, seller_id) VALUES
    ('650e8400-e29b-41d4-a716-446655440001', 'Laptop Pro 15', 'High-performance laptop with 16GB RAM', 1299.99, 50, 'Electronics', 'ACTIVE', '550e8400-e29b-41d4-a716-446655440002'),
    ('650e8400-e29b-41d4-a716-446655440002', 'Wireless Mouse', 'Ergonomic wireless mouse', 29.99, 200, 'Electronics', 'ACTIVE', '550e8400-e29b-41d4-a716-446655440002'),
    ('650e8400-e29b-41d4-a716-446655440003', 'Office Chair', 'Comfortable office chair with lumbar support', 199.99, 30, 'Furniture', 'ACTIVE', '550e8400-e29b-41d4-a716-446655440002'),
    ('650e8400-e29b-41d4-a716-446655440004', 'Desk Lamp', 'LED desk lamp with adjustable brightness', 39.99, 100, 'Furniture', 'ACTIVE', '550e8400-e29b-41d4-a716-446655440002'),
    ('650e8400-e29b-41d4-a716-446655440005', 'Notebook Set', 'Set of 5 premium notebooks', 15.99, 500, 'Stationery', 'ACTIVE', '550e8400-e29b-41d4-a716-446655440002')
ON CONFLICT (id) DO NOTHING;

-- Sample promo codes
INSERT INTO promo_codes (id, code, discount_type, discount_value, min_order_amount, max_uses, current_uses, valid_from, valid_until, active) VALUES
    ('750e8400-e29b-41d4-a716-446655440001', 'SAVE20', 'PERCENTAGE', 20.00, 100.00, 100, 0, '2026-01-01 00:00:00', '2026-12-31 23:59:59', true),
    ('750e8400-e29b-41d4-a716-446655440002', 'FIXED50', 'FIXED_AMOUNT', 50.00, 200.00, 50, 0, '2026-01-01 00:00:00', '2026-12-31 23:59:59', true),
    ('750e8400-e29b-41d4-a716-446655440003', 'WELCOME10', 'PERCENTAGE', 10.00, 50.00, 1000, 0, '2026-01-01 00:00:00', '2026-12-31 23:59:59', true)
ON CONFLICT (code) DO NOTHING;
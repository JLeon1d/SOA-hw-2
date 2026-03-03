-- Database initialization and migrations for Marketplace Backend
-- This file contains all table definitions, indexes, and constraints

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create ENUM types
CREATE TYPE user_role AS ENUM ('USER', 'SELLER', 'ADMIN');
CREATE TYPE product_status AS ENUM ('ACTIVE', 'INACTIVE', 'ARCHIVED');
CREATE TYPE order_status AS ENUM ('CREATED', 'PAYMENT_PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELED');
CREATE TYPE discount_type AS ENUM ('PERCENTAGE', 'FIXED_AMOUNT');
CREATE TYPE operation_type AS ENUM ('CREATE_ORDER', 'UPDATE_ORDER');

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'USER',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Products table
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description VARCHAR(4000),
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    category VARCHAR(100) NOT NULL,
    status product_status NOT NULL DEFAULT 'ACTIVE',
    seller_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on status for filtering (required by task)
CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);

-- Create index on category for filtering
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- Create index on seller_id for seller's product queries
CREATE INDEX IF NOT EXISTS idx_products_seller_id ON products(seller_id);

-- Promo codes table
CREATE TABLE IF NOT EXISTS promo_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(20) NOT NULL UNIQUE,
    discount_type discount_type NOT NULL,
    discount_value DECIMAL(12,2) NOT NULL CHECK (discount_value > 0),
    min_order_amount DECIMAL(12,2) NOT NULL CHECK (min_order_amount >= 0),
    max_uses INTEGER NOT NULL CHECK (max_uses > 0),
    current_uses INTEGER NOT NULL DEFAULT 0 CHECK (current_uses >= 0),
    valid_from TIMESTAMP NOT NULL,
    valid_until TIMESTAMP NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    CHECK (valid_until > valid_from),
    CHECK (current_uses <= max_uses)
);

-- Create index on code for faster lookups
CREATE INDEX IF NOT EXISTS idx_promo_codes_code ON promo_codes(code);

-- Create index on active status
CREATE INDEX IF NOT EXISTS idx_promo_codes_active ON promo_codes(active);

-- Orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status order_status NOT NULL DEFAULT 'CREATED',
    promo_code_id UUID REFERENCES promo_codes(id) ON DELETE SET NULL,
    total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0),
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on user_id for user's order queries
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);

-- Create index on status for filtering
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

-- Create composite index for checking active orders
CREATE INDEX IF NOT EXISTS idx_orders_user_status ON orders(user_id, status);

-- Order items table
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    price_at_order DECIMAL(12,2) NOT NULL CHECK (price_at_order >= 0)
);

-- Create index on order_id for faster order item lookups
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

-- Create index on product_id
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);

-- User operations table (for rate limiting)
CREATE TABLE IF NOT EXISTS user_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation_type operation_type NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create composite index for rate limiting queries
CREATE INDEX IF NOT EXISTS idx_user_operations_user_type_time ON user_operations(user_id, operation_type, created_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data for testing (optional, can be removed in production)
-- Sample users
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

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO marketplace;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO marketplace;
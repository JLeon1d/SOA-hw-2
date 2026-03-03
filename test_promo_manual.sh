#!/bin/bash

BASE_URL="http://localhost:8080"

# Register admin
echo "=== Registering admin ==="
ADMIN_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@test.com",
    "password": "password123",
    "role": "ADMIN"
  }')
echo "$ADMIN_RESPONSE"
ADMIN_TOKEN=$(echo "$ADMIN_RESPONSE" | jq -r '.access_token')
echo "Admin token: $ADMIN_TOKEN"

# Register user
echo -e "\n=== Registering user ==="
USER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@test.com",
    "password": "password123",
    "role": "USER"
  }')
echo "$USER_RESPONSE"
USER_TOKEN=$(echo "$USER_RESPONSE" | jq -r '.access_token')
echo "User token: $USER_TOKEN"

# Register seller
echo -e "\n=== Registering seller ==="
SELLER_RESPONSE=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "seller@test.com",
    "password": "password123",
    "role": "SELLER"
  }')
echo "$SELLER_RESPONSE"
SELLER_TOKEN=$(echo "$SELLER_RESPONSE" | jq -r '.access_token')
echo "Seller token: $SELLER_TOKEN"

# Create product
echo -e "\n=== Creating product ==="
PRODUCT_RESPONSE=$(curl -s -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{
    "name": "Test Product",
    "description": "Test Description",
    "price": 99.99,
    "stock": 100,
    "category": "electronics"
  }')
echo "$PRODUCT_RESPONSE"
PRODUCT_ID=$(echo "$PRODUCT_RESPONSE" | jq -r '.id')
echo "Product ID: $PRODUCT_ID"

# Create promo code
echo -e "\n=== Creating promo code ==="
# Use a time in the past to ensure it's valid NOW
VALID_FROM=$(date -u -v-1d +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "yesterday" +"%Y-%m-%dT%H:%M:%SZ")
VALID_UNTIL=$(date -u -v+30d +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d "+30 days" +"%Y-%m-%dT%H:%M:%SZ")
echo "Valid from: $VALID_FROM"
echo "Valid until: $VALID_UNTIL"

PROMO_RESPONSE=$(curl -s -X POST "$BASE_URL/promo-codes" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{
    \"code\": \"SAVE10\",
    \"discount_type\": \"PERCENTAGE\",
    \"discount_value\": 10.0,
    \"min_order_amount\": 50.0,
    \"max_uses\": 100,
    \"valid_from\": \"$VALID_FROM\",
    \"valid_until\": \"$VALID_UNTIL\"
  }")
echo "$PROMO_RESPONSE"
PROMO_ID=$(echo "$PROMO_RESPONSE" | jq -r '.id')
echo "Promo ID: $PROMO_ID"

# Try to create order with promo code
echo -e "\n=== Creating order with promo code ==="
ORDER_RESPONSE=$(curl -s -X POST "$BASE_URL/orders" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -d "{
    \"items\": [
      {
        \"product_id\": \"$PRODUCT_ID\",
        \"quantity\": 2
      }
    ],
    \"promo_code\": \"SAVE10\"
  }")
echo "$ORDER_RESPONSE"

echo -e "\n=== Done ==="
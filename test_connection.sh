#!/bin/sh

echo "=== Environment Variables ==="
echo "DATABASE_URL: $DATABASE_URL"
echo "DB_HOST: $DB_HOST"
echo "DB_PORT: $DB_PORT"
echo ""

echo "=== Network Test ==="
echo "Testing connection to postgres:5432..."
nc -zv postgres 5432 2>&1 || echo "Connection failed"
echo ""

echo "=== DNS Resolution ==="
echo "Resolving postgres hostname..."
nslookup postgres 2>&1 || getent hosts postgres 2>&1 || echo "DNS resolution failed"
echo ""

echo "=== PostgreSQL Connection Test ==="
echo "Attempting psql connection..."
PGPASSWORD=marketplace_password psql -h postgres -U marketplace -d marketplace -c "SELECT version();" 2>&1 || echo "PostgreSQL connection failed"
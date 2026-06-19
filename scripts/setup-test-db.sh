#!/bin/bash
# Setup script for test database
# Creates the purchase_api_test database for integration tests

set -e

# Configuration
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
TEST_DB_NAME="purchase_api_test"

echo "Setting up test database: $TEST_DB_NAME"
echo "Database host: $DB_HOST:$DB_PORT"

# Connect to postgres database to create test database
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "postgres" <<EOF
-- Drop existing test database if it exists
DROP DATABASE IF EXISTS "$TEST_DB_NAME";

-- Create fresh test database
CREATE DATABASE "$TEST_DB_NAME" 
  OWNER "$DB_USER"
  ENCODING 'UTF8';
EOF

echo "✓ Test database '$TEST_DB_NAME' created successfully"
echo ""
echo "You can now run integration tests with:"
echo "  go test ./tests/integration -v"
echo ""
echo "Or with custom database URL:"
echo "  TEST_DATABASE_URL='postgres://user:pass@host:5432/$TEST_DB_NAME?sslmode=disable' go test ./tests/integration -v"

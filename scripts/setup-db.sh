#!/bin/bash

# Setup database and import pricing zones data
set -e

echo "Setting up database and importing pricing zones data..."

# Database connection string
DB_DSN="${POSTGRES_DSN:-postgres://app:app@localhost:5432/payments?sslmode=disable}"

# Check if database is accessible
echo "Checking database connection..."
if ! psql "$DB_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    echo "Error: Cannot connect to database at $DB_DSN"
    echo "Please ensure PostgreSQL is running and accessible."
    echo "You can start it with: docker-compose -f docker/docker-compose.yaml up -d postgres"
    exit 1
fi

echo "Database connection successful!"

# Create the pricing_zones table
echo "Creating pricing_zones table..."
psql "$DB_DSN" -f scripts/setup-pricing-zones.sql

echo "Database setup complete!"
echo ""
echo "You can now run the application with:"
echo "  make run"
echo ""
echo "Or test the pricing zones with:"
echo "  go run ./cmd/import-pricing-zones /path/to/your/csv/file.csv"

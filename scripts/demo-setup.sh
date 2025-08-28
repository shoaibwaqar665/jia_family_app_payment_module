#!/bin/bash

# Demo script to set up database and import pricing zones data
set -e

echo "🚀 Setting up Payment Service with Dynamic Pricing Zones"
echo "========================================================"

# Check if PostgreSQL is running
echo "📋 Checking database connection..."
DB_DSN="${POSTGRES_DSN:-postgres://app:app@localhost:5432/payments?sslmode=disable}"

if ! psql "$DB_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    echo "❌ Database not accessible. Please start PostgreSQL first:"
    echo "   Option 1: Start with Docker:"
    echo "     cd docker && docker-compose up -d postgres"
    echo "   Option 2: Start local PostgreSQL service"
    echo ""
    echo "   Then run this script again."
    exit 1
fi

echo "✅ Database connection successful!"

# Create the pricing_zones table and insert data
echo "📊 Creating pricing_zones table and importing data..."
psql "$DB_DSN" -f scripts/setup-pricing-zones.sql

echo "✅ Pricing zones data imported successfully!"

# Test the import by querying some data
echo "🧪 Testing the data import..."
echo ""
echo "Sample pricing zones by zone type:"
echo "----------------------------------"

echo "Zone A (Premium - 100% pricing):"
psql "$DB_DSN" -c "SELECT country, iso_code, pricing_multiplier FROM pricing_zones WHERE zone = 'A' ORDER BY country LIMIT 5;"

echo ""
echo "Zone B (Mid-High - 70% pricing):"
psql "$DB_DSN" -c "SELECT country, iso_code, pricing_multiplier FROM pricing_zones WHERE zone = 'B' ORDER BY country LIMIT 5;"

echo ""
echo "Zone C (Mid-Low - 40% pricing):"
psql "$DB_DSN" -c "SELECT country, iso_code, pricing_multiplier FROM pricing_zones WHERE zone = 'C' ORDER BY country LIMIT 5;"

echo ""
echo "Zone D (Low-Income - 20% pricing):"
psql "$DB_DSN" -c "SELECT country, iso_code, pricing_multiplier FROM pricing_zones WHERE zone = 'D' ORDER BY country LIMIT 5;"

echo ""
echo "📈 Total pricing zones imported:"
psql "$DB_DSN" -c "SELECT COUNT(*) as total_zones FROM pricing_zones;"

echo ""
echo "🎉 Setup complete! You can now:"
echo "   1. Run the payment service: make run"
echo "   2. Test dynamic pricing with different country codes"
echo "   3. Use the pricing zones in your checkout flow"
echo ""
echo "Example usage in code:"
echo "   - US customer: 100% of base price (Zone A)"
echo "   - Indian customer: 40% of base price (Zone C)"
echo "   - Ethiopian customer: 20% of base price (Zone D)"

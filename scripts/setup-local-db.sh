#!/bin/bash

# Setup pricing zones table in your local PostgreSQL database
set -e

echo "🗄️  Setting up Pricing Zones in Local PostgreSQL Database"
echo "========================================================"

# Database connection details - MODIFY THESE FOR YOUR SETUP
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-payments}"
DB_USER="${DB_USER:-postgres}"
DB_PASSWORD="${DB_PASSWORD:-}"

# Construct connection string
if [ -n "$DB_PASSWORD" ]; then
    DB_DSN="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
else
    DB_DSN="postgresql://${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
fi

echo "📋 Database connection details:"
echo "   Host: $DB_HOST"
echo "   Port: $DB_PORT"
echo "   Database: $DB_NAME"
echo "   User: $DB_USER"
echo ""

# Test database connection
echo "🔍 Testing database connection..."
if ! psql "$DB_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    echo "❌ Cannot connect to database!"
    echo ""
    echo "Please check your database credentials and ensure PostgreSQL is running."
    echo ""
    echo "You can set environment variables:"
    echo "  export DB_HOST=localhost"
    echo "  export DB_PORT=5432"
    echo "  export DB_NAME=your_database_name"
    echo "  export DB_USER=your_username"
    echo "  export DB_PASSWORD=your_password"
    echo ""
    echo "Or modify this script directly with your credentials."
    exit 1
fi

echo "✅ Database connection successful!"

# Create the pricing_zones table
echo "📊 Creating pricing_zones table..."
psql "$DB_DSN" -f scripts/setup-pricing-zones.sql

echo "✅ Pricing zones table created and data imported!"

# Verify the data
echo "🧪 Verifying imported data..."
echo ""
echo "Total pricing zones:"
psql "$DB_DSN" -c "SELECT COUNT(*) as total_zones FROM pricing_zones;"

echo ""
echo "Sample data by zone:"
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
echo "🎉 Setup complete!"
echo ""
echo "Your pricing zones are now ready to use in the payment service."
echo "Update your config.yaml with the database connection details:"
echo "  postgres:"
echo "    dsn: \"$DB_DSN\""
echo "    max_conns: 10"

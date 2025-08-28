#!/bin/bash

# Setup pricing zones using existing config.yaml
set -e

echo "🗄️  Setting up Pricing Zones using config.yaml"
echo "=============================================="

# Extract database DSN from config.yaml
DB_DSN=$(grep "dsn:" config.yaml | sed 's/.*dsn: *"\(.*\)".*/\1/')

if [ -z "$DB_DSN" ]; then
    echo "❌ Could not find database DSN in config.yaml"
    exit 1
fi

echo "📋 Using database DSN from config.yaml"
echo "   DSN: $DB_DSN"
echo ""

# Test database connection
echo "🔍 Testing database connection..."
if ! psql "$DB_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    echo "❌ Cannot connect to database!"
    echo "Please check your database credentials in config.yaml"
    exit 1
fi

echo "✅ Database connection successful!"

# Create the pricing_zones table
echo "📊 Creating pricing_zones table and importing data..."
psql "$DB_DSN" -f scripts/setup-pricing-zones.sql

echo "✅ Setup complete!"

# Show summary
echo ""
echo "📈 Summary:"
psql "$DB_DSN" -c "SELECT zone, COUNT(*) as countries FROM pricing_zones GROUP BY zone ORDER BY zone;"

echo ""
echo "🎉 Your pricing zones database is ready!"

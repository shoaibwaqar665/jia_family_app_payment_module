#!/bin/bash

# Interactive setup script for pricing zones database
set -e

echo "🗄️  Interactive Database Setup for Pricing Zones"
echo "================================================"
echo ""

# Prompt for database credentials
read -p "Database host [localhost]: " DB_HOST
DB_HOST=${DB_HOST:-localhost}

read -p "Database port [5432]: " DB_PORT
DB_PORT=${DB_PORT:-5432}

read -p "Database name: " DB_NAME
if [ -z "$DB_NAME" ]; then
    echo "❌ Database name is required!"
    exit 1
fi

read -p "Database username: " DB_USER
if [ -z "$DB_USER" ]; then
    echo "❌ Database username is required!"
    exit 1
fi

read -s -p "Database password (press Enter if no password): " DB_PASSWORD
echo ""

# Construct connection string
if [ -n "$DB_PASSWORD" ]; then
    DB_DSN="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
else
    DB_DSN="postgresql://${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
fi

echo ""
echo "📋 Testing connection to: $DB_HOST:$DB_PORT/$DB_NAME as $DB_USER"

# Test database connection
if ! psql "$DB_DSN" -c "SELECT 1;" >/dev/null 2>&1; then
    echo "❌ Cannot connect to database!"
    echo "Please check your credentials and ensure PostgreSQL is running."
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
echo ""
echo "Update your config.yaml with:"
echo "  postgres:"
echo "    dsn: \"$DB_DSN\""
echo "    max_conns: 10"

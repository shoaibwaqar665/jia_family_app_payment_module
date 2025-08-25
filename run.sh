#!/bin/bash

# Payment Service Runner Script
echo "🚀 Starting Payment Service..."

# Check if .env file exists
if [ ! -f .env ]; then
    echo "❌ .env file not found!"
    echo "📝 Creating .env file with your credentials..."
    
    cat > .env << 'EOF'
# Database Configuration
POSTGRES_DSN=postgres://neondb_owner:sLdJyF0w2Unv@ep-wild-wave-a1nsn7ul.ap-southeast-1.aws.neon.tech:5432/jia_family_app?sslmode=require
POSTGRES_MAX_CONNS=10

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_DB=0
REDIS_PASSWORD=

# gRPC Configuration
GRPC_ADDRESS=:8081

# Application Configuration
APP_NAME=PaymentService
ENV=development

# Authentication (placeholder)
AUTH_PUBLIC_KEY_PEM=""

# Billing Provider
BILLING_PROVIDER=stripe
STRIPE_SECRET=
STRIPE_PUBLISHABLE_KEY=

# Events Configuration
EVENTS_PROVIDER=noop
EVENTS_BROKERS=["localhost:9092"]
EVENTS_TOPIC=entitlement-events
EOF

    echo "✅ .env file created!"
fi

# Check if Redis is running
echo "🔍 Checking Redis..."
if ! docker ps | grep -q redis; then
    echo "📦 Starting Redis container..."
    docker run -d --name redis-test -p 6379:6379 redis:7 > /dev/null 2>&1
    echo "✅ Redis started!"
else
    echo "✅ Redis is already running!"
fi

# Wait a moment for Redis to be ready
sleep 2

# Check if Go modules are tidy
echo "🔧 Checking Go modules..."
go mod tidy

# Run the payment service
echo "🚀 Starting Payment Service..."
echo "📍 Service will be available at: localhost:8081"
echo "🔑 Health check: grpc://localhost:8081/grpc.health.v1.Health/Check"
echo ""
echo "Press Ctrl+C to stop the service"
echo ""

go run ./cmd/paymentservice

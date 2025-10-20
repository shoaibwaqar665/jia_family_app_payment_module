# Payment Module Integration Guide

## 🎯 Overview

This guide provides comprehensive instructions for integrating the Payment Module into the Jia Family App system. The payment module is a complete gRPC-based microservice that handles payment processing, subscription management, entitlement checking, and usage tracking.

## 📋 Table of Contents

1. [System Architecture](#system-architecture)
2. [Integration Points](#integration-points)
3. [Setup Instructions](#setup-instructions)
4. [Configuration Guide](#configuration-guide)
5. [API Integration](#api-integration)
6. [Database Integration](#database-integration)
7. [Event System Integration](#event-system-integration)
8. [Monitoring & Health Checks](#monitoring--health-checks)
9. [Deployment Guide](#deployment-guide)
10. [Developer Questions](#developer-questions)

---

## 🏗️ System Architecture

### High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client App    │    │   Admin Panel   │    │  Webhook Source │
│                 │    │                 │    │   (Stripe)      │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │ gRPC                 │ gRPC                 │ HTTP
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Payment Service                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Transport │  │   Use Case  │  │   Domain    │            │
│  │   Layer     │  │   Layer     │  │   Layer     │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ Repository  │  │   Cache     │  │   Events    │            │
│  │   Layer     │  │   (Redis)   │  │   (Kafka)   │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   PostgreSQL    │    │     Redis        │    │     Kafka       │
│   Database      │    │     Cache        │    │   Event Bus     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Core Components

1. **Transport Layer** (`internal/payment/transport/`)
   - gRPC service implementation
   - Request/response transformation
   - Error handling and metrics

2. **Use Case Layer** (`internal/payment/usecase/`)
   - Business logic orchestration
   - Payment processing
   - Entitlement management
   - Checkout session creation

3. **Domain Layer** (`internal/payment/domain/`)
   - Core business entities
   - Domain validation rules
   - Business invariants

4. **Repository Layer** (`internal/payment/repo/`)
   - Data access abstraction
   - PostgreSQL implementation
   - Query optimization

---

## 🔗 Integration Points

### 1. Main Application Bootstrap

The payment service integrates through the main application bootstrap process:

**File**: `internal/app/app.go`
```go
func BootstrapAndServe(ctx context.Context) error {
    // 1. Load configuration
    cfg, err := config.Load("config.yaml")
    
    // 2. Initialize dependencies (DB, Redis, Cache)
    // 3. Initialize repositories
    // 4. Initialize billing provider (Stripe)
    // 5. Initialize event publisher (Kafka)
    // 6. Initialize use cases
    // 7. Initialize payment service
    // 8. Register with gRPC server
    // 9. Start health monitoring
    // 10. Start gRPC server
}
```

### 2. gRPC Server Registration

**File**: `internal/app/server/grpc_server.go`
```go
func (s *GRPCServer) RegisterPaymentService(paymentService paymentv1.PaymentServiceServer) {
    paymentv1.RegisterPaymentServiceServer(s.server, paymentService)
}
```

### 3. Service Dependencies

The payment service requires these dependencies:
- **PostgreSQL Database**: For data persistence
- **Redis**: For caching and rate limiting
- **Stripe**: For payment processing
- **Kafka**: For event publishing (optional)

---

## 🚀 Setup Instructions

### Prerequisites

1. **Go 1.21+**
2. **PostgreSQL 13+**
3. **Redis 6+**
4. **Stripe Account** (for payment processing)
5. **Kafka** (optional, for events)

### 1. Database Setup

```bash
# Run database migrations
make migrate-up

# Or manually:
psql -d jia_family_app -f migrations/0001_init.sql
psql -d jia_family_app -f migrations/0002_seed_plans.sql
psql -d jia_family_app -f migrations/0003_pricing_zones.sql
psql -d jia_family_app -f migrations/0004_payments.sql
psql -d jia_family_app -f migrations/0005_update_amount_to_decimal.sql
psql -d jia_family_app -f migrations/0006_subscriptions.sql
psql -d jia_family_app -f migrations/0007_usage.sql
```

### 2. Environment Setup

Create a `.env` file or set environment variables:

```bash
# Database
export POSTGRES_DSN="postgres://user:password@localhost:5432/jia_family_app"
export POSTGRES_MAX_CONNS=10

# Redis
export REDIS_ADDR="localhost:6379"
export REDIS_DB=0
export REDIS_PASSWORD=""

# Stripe
export STRIPE_SECRET="sk_test_..."
export STRIPE_PUBLISHABLE_KEY="pk_test_..."
export STRIPE_WEBHOOK_SECRET="whsec_..."

# Kafka (optional)
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="payments"

# Application
export GRPC_ADDRESS=":8081"
export LOG_LEVEL="info"
```

### 3. Build and Run

```bash
# Build the service
go build -o payment-service ./cmd/paymentservice

# Run the service
./payment-service

# Or run directly
go run ./cmd/paymentservice
```

---

## ⚙️ Configuration Guide

### Configuration File Structure

**File**: `config.yaml`

```yaml
app_name: "payment-service"

grpc:
  address: ":8081"

postgres:
  dsn: "postgres://user:password@localhost:5432/jia_family_app"
  max_conns: 10

redis:
  addr: "localhost:6379"
  db: 0
  password: ""

auth:
  public_key_pem: ""

billing:
  provider: "stripe"
  stripe_secret: "sk_test_..."
  stripe_publishable: "pk_test_..."
  stripe_webhook_secret: "whsec_..."

events:
  provider: "kafka"  # or "noop"
  brokers: ["localhost:9092"]
  topic: "payments"

log:
  level: "info"
```

### Environment Variable Override

The configuration supports environment variable overrides:

```yaml
# In config.yaml
postgres:
  dsn: "${POSTGRES_DSN}"
  max_conns: ${POSTGRES_MAX_CONNS}

redis:
  addr: "${REDIS_ADDR}"
  db: ${REDIS_DB}
  password: "${REDIS_PASSWORD}"
```

---

## 🔌 API Integration

### gRPC Service Definition

**File**: `api/payment/v1/payment_service.proto`

Key services:
- `CreatePayment` - Create a new payment
- `GetPayment` - Retrieve payment by ID
- `UpdatePaymentStatus` - Update payment status
- `CreateCheckoutSession` - Create Stripe checkout session
- `ProcessWebhook` - Process Stripe webhooks
- `CheckEntitlement` - Check user feature access
- `BulkCheckEntitlements` - Batch entitlement checking
- `ListPricingZones` - Get pricing zones

### Client Integration Example

```go
package main

import (
    "context"
    "google.golang.org/grpc"
    paymentv1 "github.com/jia-app/paymentservice/api/payment/v1"
)

func main() {
    // Connect to payment service
    conn, err := grpc.Dial("localhost:8081", grpc.WithInsecure())
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := paymentv1.NewPaymentServiceClient(conn)

    // Create checkout session
    resp, err := client.CreateCheckoutSession(context.Background(), &paymentv1.CreateCheckoutSessionRequest{
        PlanId:      "premium-plan",
        UserId:      "user-123",
        CountryCode: "US",
        BasePrice:   29.99,
        Currency:    "usd",
        SuccessUrl:  "https://app.example.com/success",
        CancelUrl:   "https://app.example.com/cancel",
    })

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Checkout URL: %s\n", resp.Url)
}
```

---

## 🗄️ Database Integration

### Database Schema

The payment module uses these main tables:

1. **plans** - Subscription plans
2. **pricing_zones** - Dynamic pricing by country
3. **payments** - Payment records
4. **subscriptions** - User subscriptions
5. **entitlements** - Feature access control
6. **usage** - Usage tracking

### Repository Pattern

```go
// Repository interfaces
type PaymentRepository interface {
    CreatePayment(ctx context.Context, payment *domain.Payment) error
    GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
    UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string) error
}

type EntitlementRepository interface {
    CreateEntitlement(ctx context.Context, entitlement *domain.Entitlement) error
    GetUserEntitlements(ctx context.Context, userID string) ([]*domain.Entitlement, error)
    CheckEntitlement(ctx context.Context, userID, featureCode string) (*domain.Entitlement, error)
}
```

### SQLC Integration

The module uses SQLC for type-safe SQL queries:

```bash
# Generate Go code from SQL
sqlc generate -f sqlc.yaml
```

---

## 📡 Event System Integration

### Event Publishing

The payment service publishes events for:
- Payment success/failure
- Subscription changes
- Entitlement updates
- Usage tracking

```go
// Event publisher interface
type EntitlementPublisher interface {
    PublishEntitlementGranted(ctx context.Context, event EntitlementGrantedEvent) error
    PublishEntitlementRevoked(ctx context.Context, event EntitlementRevokedEvent) error
}

// Usage in use case
func (u *EntitlementUseCase) GrantEntitlement(ctx context.Context, req *GrantEntitlementRequest) error {
    // ... business logic ...
    
    // Publish event
    event := events.EntitlementGrantedEvent{
        UserID:      req.UserID,
        FeatureCode: req.FeatureCode,
        PlanID:      req.PlanID,
        GrantedAt:   time.Now(),
    }
    
    return u.publisher.PublishEntitlementGranted(ctx, event)
}
```

### Event Consumption

Other services can consume these events:

```go
// Example event consumer
func (s *UserService) HandleEntitlementGranted(ctx context.Context, event EntitlementGrantedEvent) error {
    // Update user permissions
    // Send notification
    // Update UI state
    return nil
}
```

---

## 📊 Monitoring & Health Checks

### Health Check Endpoint

The service provides gRPC health checks:

```go
// Health check client
conn, err := grpc.Dial("localhost:8081", grpc.WithInsecure())
client := healthpb.NewHealthClient(conn)

resp, err := client.Check(context.Background(), &healthpb.HealthCheckRequest{
    Service: "", // Empty for overall health
})

switch resp.Status {
case healthpb.HealthCheckResponse_SERVING:
    // Service is healthy
case healthpb.HealthCheckResponse_NOT_SERVING:
    // Service is unhealthy
}
```

### Metrics Collection

The service collects metrics for:
- Payment processing latency
- Entitlement check performance
- Webhook processing success/failure
- Cache hit/miss ratios

### Logging

Structured logging with Zap:

```go
log.Info(ctx, "Payment processed successfully",
    zap.String("payment_id", payment.ID.String()),
    zap.Float64("amount", payment.Amount),
    zap.String("currency", payment.Currency))
```

---

## 🚀 Deployment Guide

### Docker Deployment

**File**: `docker/Dockerfile`

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o payment-service ./cmd/paymentservice

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/payment-service .
COPY --from=builder /app/config.yaml .
CMD ["./payment-service"]
```

### Docker Compose

**File**: `docker/docker-compose.yaml`

```yaml
version: '3.8'
services:
  payment-service:
    build: .
    ports:
      - "8081:8081"
    environment:
      - POSTGRES_DSN=postgres://user:password@postgres:5432/jia_family_app
      - REDIS_ADDR=redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: jia_family_app
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:6-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payment-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: payment-service
  template:
    metadata:
      labels:
        app: payment-service
    spec:
      containers:
      - name: payment-service
        image: payment-service:latest
        ports:
        - containerPort: 8081
        env:
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef:
              name: payment-secrets
              key: postgres-dsn
        - name: STRIPE_SECRET
          valueFrom:
            secretKeyRef:
              name: payment-secrets
              key: stripe-secret
```

---

## ❓ Developer Questions

### For Frontend Developers

1. **Authentication Integration**
   - How do you handle user authentication tokens in gRPC calls?
   - What's the expected format for user IDs and family IDs?
   - How should the frontend handle payment success/failure flows?

2. **UI/UX Questions**
   - What's the expected user flow for subscription upgrades?
   - How should pricing be displayed for different countries?
   - What error messages should be shown for payment failures?

3. **Feature Access Control**
   - How do you want to handle feature access checks in the UI?
   - Should there be real-time updates when entitlements change?
   - How should usage limits be displayed to users?

### For Backend Developers

1. **Service Integration**
   - How should other services authenticate with the payment service?
   - What's the expected service discovery mechanism?
   - How should service-to-service communication be handled?

2. **Data Consistency**
   - How should payment data be synchronized with user data?
   - What's the strategy for handling eventual consistency?
   - How should failed payments be retried?

3. **Event Handling**
   - What events should other services listen to?
   - How should event ordering be handled?
   - What's the retry strategy for failed event processing?

### For DevOps/Infrastructure

1. **Scaling Considerations**
   - What's the expected load for payment processing?
   - How should the service scale horizontally?
   - What are the resource requirements?

2. **Security**
   - How should sensitive data (Stripe keys) be managed?
   - What's the network security model?
   - How should webhook endpoints be secured?

3. **Monitoring**
   - What metrics are most important for monitoring?
   - How should alerts be configured?
   - What's the log retention policy?

### For Product/Design

1. **Business Logic**
   - What are the subscription plan requirements?
   - How should pricing zones be configured?
   - What features require entitlement checking?

2. **User Experience**
   - What's the expected checkout flow?
   - How should subscription management work?
   - What notifications should users receive?

3. **Compliance**
   - What payment regulations need to be considered?
   - How should data privacy be handled?
   - What audit requirements exist?

---

## 🔧 Troubleshooting

### Common Issues

1. **Database Connection Issues**
   ```bash
   # Check database connectivity
   psql -h localhost -p 5432 -U user -d jia_family_app -c "SELECT 1;"
   ```

2. **Redis Connection Issues**
   ```bash
   # Check Redis connectivity
   redis-cli ping
   ```

3. **Stripe Webhook Issues**
   - Verify webhook secret configuration
   - Check webhook endpoint URL
   - Review Stripe dashboard for webhook logs

4. **gRPC Connection Issues**
   ```bash
   # Test gRPC connectivity
   grpcurl -plaintext localhost:8081 list
   ```

### Debug Mode

Enable debug logging:

```yaml
log:
  level: "debug"
```

### Health Check Debugging

```bash
# Check service health
grpcurl -plaintext localhost:8081 grpc.health.v1.Health/Check
```

---

## 📚 Additional Resources

- [Payment Service Architecture Guide](./PAYMENT_SERVICE_ARCHITECTURE_GUIDE.md)
- [API Documentation](./api/payment/v1/payment_service.proto)
- [Database Schema](./migrations/)
- [Testing Guide](./TESTING_GUIDE.md)
- [Webhook Processing Guide](./WEBHOOK_PROCESSING_GUIDE.md)

---

## 🤝 Contributing

When contributing to the payment module:

1. Follow the existing code structure and patterns
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Ensure backward compatibility
5. Follow the established error handling patterns

For questions or issues, please refer to the developer questions section above or create an issue in the repository.

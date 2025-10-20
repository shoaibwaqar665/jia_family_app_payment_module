# Payment Service Architecture Guide

## Table of Contents
1. [Overview](#overview)
2. [Architecture Overview](#architecture-overview)
3. [Directory Structure](#directory-structure)
4. [Core Components](#core-components)
5. [Data Flow](#data-flow)
6. [Webhook Structure & Flow](#webhook-structure--flow)
7. [Dependencies](#dependencies)
8. [API Endpoints](#api-endpoints)
9. [Database Schema](#database-schema)
10. [Configuration](#configuration)
11. [Monitoring & Metrics](#monitoring--metrics)
12. [Deployment Guide](#deployment-guide)

## Overview

The Payment Service is a comprehensive microservice built in Go that handles payment processing, subscription management, entitlement checking, and usage tracking. It's designed as a gRPC service with comprehensive monitoring, circuit breakers, and rate limiting.

### Key Features
- **Payment Processing**: Stripe integration for payment handling
- **Subscription Management**: Lifecycle management (active, past_due, suspended, cancelled)
- **Entitlement System**: Feature access control and bulk checking
- **Usage Tracking**: Quota management and usage-based billing
- **Webhook Processing**: Secure Stripe webhook handling
- **Circuit Breakers**: Resilience patterns for external services
- **Rate Limiting**: Redis-based rate limiting
- **Monitoring**: Comprehensive metrics and health checks
- **Dunning Management**: Failed payment retry logic

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Applications                      │
└─────────────────────┬───────────────────────────────────────────┘
                      │ gRPC
┌─────────────────────▼───────────────────────────────────────────┐
│                    gRPC Server                                  │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │   Interceptors  │ │   Middleware    │ │   Health Check  │   │
│  │  - Auth         │ │  - Metrics      │ │  - Monitoring   │   │
│  │  - Logging      │ │  - Rate Limit   │ │  - Circuit Br.  │   │
│  │  - Recovery     │ │  - Circuit Br.  │ │                 │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                 Transport Layer                                  │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Payment Service │ │ Webhook Handler │ │ Health Service  │   │
│  │ - CreatePayment │ │ - Validation    │ │ - Status Check  │   │
│  │ - CheckEntitle  │ │ - Parsing       │ │ - Metrics       │   │
│  │ - Bulk Check    │ │ - Processing    │ │                 │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                  Use Case Layer                                 │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Payment UseCase │ │ Entitlement UC  │ │ Subscription UC │   │
│  │ - Process       │ │ - Check         │ │ - Lifecycle     │   │
│  │ - Validate      │ │ - Bulk Check    │ │ - Management    │   │
│  │ - Create        │ │ - Cache         │ │                 │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Usage Tracker   │ │ Dunning Manager │ │ Checkout UC     │   │
│  │ - Track Usage   │ │ - Retry Logic   │ │ - Session Mgmt  │   │
│  │ - Quota Check   │ │ - Escalation    │ │ - Webhook Appl. │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                  Repository Layer                               │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Payment Repo    │ │ Entitlement Repo│ │ Subscription Rep│   │
│  │ - CRUD Ops      │ │ - CRUD Ops      │ │ - CRUD Ops      │   │
│  │ - Status Update │ │ - Status Update │ │ - Lifecycle     │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Usage Repo      │ │ Plan Repo       │ │ Pricing Zone    │   │
│  │ - Track Usage   │ │ - Plan Mgmt     │ │ - Dynamic Price │   │
│  │ - History       │ │ - Features      │ │ - Zone Mgmt     │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                  External Services                              │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Stripe API      │ │ Redis Cache     │ │ PostgreSQL DB   │   │
│  │ - Payments      │ │ - Entitlements  │ │ - All Data      │   │
│  │ - Subscriptions │ │ - Rate Limiting │ │ - Persistence   │   │
│  │ - Webhooks      │ │ - Sessions      │ │                 │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
jia_family_app/
├── api/                                    # API Definitions
│   └── payment/v1/
│       ├── payment_service.proto           # gRPC service definition
│       ├── payment_service.pb.go           # Generated Go code
│       └── payment_service_grpc.pb.go      # Generated gRPC code
├── cmd/
│   └── paymentservice/
│       └── main.go                         # Application entry point
├── internal/
│   ├── app/                                # Application layer
│   │   ├── app.go                          # Dependency injection
│   │   └── server/
│   │       ├── grpc_server.go              # gRPC server setup
│   │       └── interceptors/               # gRPC interceptors
│   ├── billing/                            # Billing providers
│   │   └── stripebp/
│   │       └── adapter.go                  # Stripe adapter
│   ├── payment/                            # Payment domain
│   │   ├── domain/
│   │   │   └── models.go                   # Domain models
│   │   ├── repo/                           # Repository interfaces
│   │   │   ├── interfaces.go               # Repository contracts
│   │   │   ├── postgres/                   # PostgreSQL implementation
│   │   │   │   ├── store.go                # Main store
│   │   │   │   ├── queries/                # SQL queries
│   │   │   │   └── pgstore/                # Generated SQL code
│   │   │   └── subscription.go             # Subscription repo
│   │   ├── subscription/                   # Subscription management
│   │   │   └── lifecycle.go                # Lifecycle logic
│   │   ├── transport/                      # Transport layer
│   │   │   └── grpc.go                     # gRPC handlers
│   │   ├── usecase/                        # Business logic
│   │   │   ├── bulk_entitlements.go        # Bulk entitlement checking
│   │   │   ├── dunning_manager.go          # Dunning management
│   │   │   ├── dunning_scheduler.go        # Dunning scheduling
│   │   │   └── usage_tracker.go            # Usage tracking
│   │   └── webhook/                        # Webhook processing
│   │       ├── parser.go                   # Webhook parsing
│   │       └── validator.go                # Webhook validation
│   └── shared/                             # Shared components
│       ├── cache/                          # Caching layer
│       ├── circuitbreaker/                 # Circuit breaker pattern
│       ├── config/                         # Configuration
│       ├── events/                         # Event publishing
│       ├── health/                         # Health checks
│       ├── log/                            # Logging
│       ├── metrics/                        # Metrics collection
│       └── ratelimit/                      # Rate limiting
├── migrations/                             # Database migrations
│   ├── 0006_subscriptions.sql
│   ├── 0006_subscriptions.down.sql
│   ├── 0007_usage.sql
│   └── 0007_usage.down.sql
├── config.yaml                            # Configuration file
├── go.mod                                 # Go module definition
└── go.sum                                 # Go module checksums
```

## Core Components

### 1. Transport Layer (`internal/payment/transport/`)

**Purpose**: Handles gRPC communication and request/response transformation

**Key Files**:
- `grpc.go`: Main gRPC service implementation

**Responsibilities**:
- Convert protobuf requests to domain models
- Call appropriate use cases
- Convert domain responses to protobuf
- Handle errors and status codes
- Collect metrics for each operation

### 2. Use Case Layer (`internal/payment/usecase/`)

**Purpose**: Contains business logic and orchestrates domain operations

**Key Components**:
- **PaymentUseCase**: Payment processing logic
- **EntitlementUseCase**: Feature access control
- **BulkEntitlementUseCase**: Batch entitlement checking
- **CheckoutUseCase**: Checkout session management
- **UsageTracker**: Usage tracking and quota management
- **DunningManager**: Failed payment retry logic
- **DunningScheduler**: Automated retry scheduling

### 3. Repository Layer (`internal/payment/repo/`)

**Purpose**: Data access abstraction and persistence

**Key Components**:
- **PaymentRepository**: Payment CRUD operations
- **EntitlementRepository**: Entitlement management
- **SubscriptionRepository**: Subscription lifecycle
- **UsageRepository**: Usage tracking
- **PlanRepository**: Plan management
- **PricingZoneRepository**: Dynamic pricing

### 4. Domain Layer (`internal/payment/domain/`)

**Purpose**: Core business entities and rules

**Key Models**:
- **Payment**: Payment transaction data
- **Entitlement**: Feature access permissions
- **Subscription**: Subscription lifecycle data
- **Usage**: Usage tracking records
- **Plan**: Subscription plans
- **PricingZone**: Dynamic pricing zones

## Data Flow

### Payment Processing Flow

```
1. Client Request (gRPC)
   ↓
2. gRPC Interceptors (Auth, Logging, Metrics, Rate Limiting)
   ↓
3. Transport Layer (grpc.go)
   - Validate request
   - Convert protobuf to domain model
   ↓
4. Use Case Layer
   - Business logic validation
   - Call billing provider (Stripe)
   ↓
5. Repository Layer
   - Persist payment record
   - Update related entities
   ↓
6. External Services
   - Stripe API call
   - Database persistence
   ↓
7. Response
   - Convert domain model to protobuf
   - Return to client
```

### Entitlement Checking Flow

```
1. Client Request (CheckEntitlement)
   ↓
2. Transport Layer
   - Extract user_id and feature_code
   ↓
3. EntitlementUseCase
   - Check cache first (Redis)
   - If cache miss, query database
   - Update cache with result
   ↓
4. Repository Layer
   - Query entitlements table
   - Apply business rules (expiry, status)
   ↓
5. Response
   - Return entitlement status
   - Include metadata (expiry, plan info)
```

## Webhook Structure & Flow

### Webhook Processing Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Stripe Webhook                          │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTP POST
┌─────────────────────▼───────────────────────────────────────────┐
│                 gRPC Server                                     │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Rate Limiting   │ │ Auth Validation │ │ Metrics Collect │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│              PaymentSuccessWebhook Handler                       │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Signature Valid │ │ Payload Parse   │ │ Business Logic  │   │
│  │ - HMAC Verify   │ │ - Event Extract │ │ - Apply Changes │   │
│  │ - Timestamp     │ │ - Data Extract  │ │ - Update State  │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                Webhook Components                               │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Validator        │ │ Parser          │ │ CheckoutUseCase │   │
│  │ - HMAC-SHA256    │ │ - Event Types   │ │ - ApplyWebhook  │   │
│  │ - Timestamp      │ │ - Data Extract  │ │ - Update Entitle│   │
│  │ - Replay Protect │ │ - Normalize     │ │ - Notify Events │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                Database Updates                                 │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Entitlements    │ │ Subscriptions   │ │ Payments        │   │
│  │ - Grant Access  │ │ - Update Status │ │ - Record Trans  │   │
│  │ - Set Expiry    │ │ - Renew Period  │ │ - Update Status │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Webhook Security Flow

```
1. Stripe sends webhook with signature header
   ↓
2. Validator.ValidateStripeWebhook()
   - Extract timestamp and signature from header
   - Check timestamp (prevent replay attacks)
   - Compute HMAC-SHA256 signature
   - Compare with provided signature
   ↓
3. If valid, proceed to parsing
   ↓
4. Parser.ParseStripeWebhook()
   - Parse JSON payload
   - Extract event type
   - Extract relevant data based on event type
   - Normalize to WebhookResult structure
   ↓
5. Apply webhook result
   - Update entitlements
   - Update subscriptions
   - Record payment
   - Publish events
```

### Supported Webhook Events

| Event Type | Description | Actions |
|------------|-------------|---------|
| `checkout.session.completed` | Payment successful | Grant entitlements, create subscription |
| `invoice.payment_succeeded` | Recurring payment | Renew subscription, extend entitlements |
| `charge.succeeded` | One-time payment | Grant entitlements |
| `customer.subscription.created` | New subscription | Create subscription record |
| `customer.subscription.updated` | Subscription changed | Update subscription status |
| `customer.subscription.deleted` | Subscription cancelled | Revoke entitlements |

### Webhook Data Structure

```go
type WebhookResult struct {
    EventType      string                 `json:"event_type"`
    UserID         string                 `json:"user_id"`
    FamilyID       *string                `json:"family_id,omitempty"`
    FeatureCode    string                 `json:"feature_code"`
    PlanID         uuid.UUID              `json:"plan_id"`
    PlanIDString   string                 `json:"plan_id_string"`
    SubscriptionID string                 `json:"subscription_id"`
    Amount         int64                  `json:"amount"`
    Currency       string                 `json:"currency"`
    Status         string                 `json:"status"`
    ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
}
```

## Dependencies

### External Dependencies

| Service | Purpose | Integration Method |
|---------|---------|-------------------|
| **Stripe** | Payment processing | REST API + Webhooks |
| **PostgreSQL** | Data persistence | SQL queries via sqlc |
| **Redis** | Caching & Rate limiting | Redis client |
| **Prometheus** | Metrics collection | Prometheus client |

### Internal Dependencies

```
┌─────────────────────────────────────────────────────────────────┐
│                    Dependency Graph                             │
│                                                                 │
│  Transport Layer                                                │
│  ├── Use Cases                                                  │
│  │   ├── Repositories                                          │
│  │   │   ├── Database (PostgreSQL)                             │
│  │   │   └── Cache (Redis)                                     │
│  │   ├── Cache Client                                          │
│  │   ├── Event Publishers                                      │
│  │   └── Billing Provider (Stripe)                             │
│  ├── Metrics Collector                                         │
│  └── Configuration                                             │
│                                                                 │
│  gRPC Server                                                    │
│  ├── Interceptors                                              │
│  │   ├── Auth                                                  │
│  │   ├── Logging                                               │
│  │   ├── Metrics                                               │
│  │   ├── Rate Limiting                                         │
│  │   └── Circuit Breaker                                       │
│  ├── Health Check Service                                      │
│  └── Reflection Service                                        │
└─────────────────────────────────────────────────────────────────┘
```

### Dependency Injection Flow

```go
// 1. Load Configuration
cfg := config.Load("config.yaml")

// 2. Initialize External Services
dbPool := pgxpool.NewWithConfig(ctx, dbConfig)
redisClient := redis.NewClient(&redis.Options{...})
billingProvider := stripebp.NewAdapter(cfg)

// 3. Initialize Repositories
repo := postgres.NewStoreWithPool(dbPool)

// 4. Initialize Use Cases
paymentUseCase := usecase.NewPaymentUseCase(repo.Payment())
entitlementUseCase := usecase.NewEntitlementUseCase(repo.Entitlement(), cacheClient, eventPublisher)

// 5. Initialize Transport Layer
paymentService := transport.NewPaymentService(cfg, paymentUseCase, entitlementUseCase, ...)

// 6. Initialize gRPC Server
grpcServer := server.NewGRPCServer(cfg, dbPool, redisClient)
grpcServer.RegisterPaymentService(paymentService)
```

## API Endpoints

### gRPC Service: `payment.v1.PaymentService`

| Method | Purpose | Input | Output |
|--------|---------|-------|--------|
| `CreatePayment` | Create new payment | `CreatePaymentRequest` | `CreatePaymentResponse` |
| `GetPayment` | Retrieve payment | `GetPaymentRequest` | `GetPaymentResponse` |
| `UpdatePaymentStatus` | Update payment status | `UpdatePaymentStatusRequest` | `UpdatePaymentStatusResponse` |
| `GetPaymentsByCustomer` | List customer payments | `GetPaymentsByCustomerRequest` | `GetPaymentsByCustomerResponse` |
| `ListPayments` | List all payments | `ListPaymentsRequest` | `ListPaymentsResponse` |
| `CreateCheckoutSession` | Create Stripe checkout | `CreateCheckoutSessionRequest` | `CreateCheckoutSessionResponse` |
| `CheckEntitlement` | Check feature access | `CheckEntitlementRequest` | `CheckEntitlementResponse` |
| `BulkCheckEntitlements` | Batch entitlement check | `BulkCheckEntitlementsRequest` | `BulkCheckEntitlementsResponse` |
| `ListEntitlements` | List user entitlements | `ListEntitlementsRequest` | `ListEntitlementsResponse` |

### HTTP Endpoints (Health & Metrics)

| Endpoint | Purpose | Method |
|----------|---------|--------|
| `/health` | Comprehensive health check | GET |
| `/health/ready` | Readiness probe | GET |
| `/health/live` | Liveness probe | GET |
| `/metrics` | Prometheus metrics | GET |
| `/status` | Detailed service status | GET |

## Database Schema

### Core Tables

#### Payments Table
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount NUMERIC(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(50) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    customer_id VARCHAR(255) NOT NULL,
    order_id VARCHAR(255),
    description TEXT,
    external_payment_id VARCHAR(255),
    failure_reason TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

#### Entitlements Table
```sql
CREATE TABLE entitlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    family_id VARCHAR(255),
    feature_code VARCHAR(255) NOT NULL,
    plan_id VARCHAR(255) NOT NULL,
    subscription_id VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    granted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    usage_limits JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

#### Subscriptions Table
```sql
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    family_id VARCHAR(255),
    plan_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL,
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    external_subscription_id VARCHAR(255) NOT NULL UNIQUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

#### Usage Records Table
```sql
CREATE TABLE usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    family_id VARCHAR(255),
    feature_code VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255) NOT NULL,
    resource_size BIGINT NOT NULL,
    operation VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

## Configuration

### Configuration Structure (`config.yaml`)

```yaml
app:
  name: "payment-service"
  version: "1.0.0"
  environment: "development"

grpc:
  address: ":8080"
  max_connection_idle: "300s"
  max_connection_age: "300s"
  max_connection_age_grace: "5s"
  time: "300s"
  timeout: "5s"

postgres:
  dsn: "postgres://user:password@localhost:5432/paymentservice"
  max_conns: 25
  min_conns: 5
  max_conn_lifetime: "1h"
  max_conn_idle_time: "30m"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 10
  min_idle_conns: 5

billing:
  provider: "stripe"
  stripe_secret: "sk_test_..."
  stripe_publishable: "pk_test_..."
  stripe_webhook_secret: "whsec_..."

events:
  provider: "kafka"
  topic: "payment-events"
  brokers: ["localhost:9092"]

log:
  level: "info"
  format: "json"
  output: "stdout"

metrics:
  enabled: true
  port: 9090
  path: "/metrics"

health:
  enabled: true
  port: 8081
  path: "/health"
```

## Monitoring & Metrics

### Metrics Collected

#### Payment Metrics
- `payment_total`: Total payment requests
- `payment_success_total`: Successful payments
- `payment_failed_total`: Failed payments
- `payment_duration_seconds`: Payment processing time
- `payment_amount_dollars`: Payment amounts

#### Entitlement Metrics
- `entitlement_checks_total`: Entitlement check requests
- `entitlement_cache_hits_total`: Cache hits
- `entitlement_cache_misses_total`: Cache misses
- `entitlement_check_duration_seconds`: Check duration

#### System Metrics
- `db_connections_active`: Active database connections
- `db_query_duration_seconds`: Database query time
- `cache_hits_total`: Cache operation hits
- `cache_misses_total`: Cache operation misses
- `circuit_breaker_state`: Circuit breaker status

### Health Checks

#### Component Health Status
- **Database**: Connection pool status, query performance
- **Redis**: Connection status, operation latency
- **Stripe**: API availability, response time
- **Circuit Breakers**: State monitoring, failure rates

#### Health Endpoints
- `/health`: Comprehensive health check
- `/health/ready`: Kubernetes readiness probe
- `/health/live`: Kubernetes liveness probe

## Deployment Guide

### Prerequisites
- Go 1.23+
- PostgreSQL 13+
- Redis 6+
- Stripe account
- Docker (optional)

### Local Development

1. **Setup Database**
```bash
# Start PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_DB=paymentservice \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 postgres:13

# Run migrations
psql -h localhost -U user -d paymentservice -f migrations/0006_subscriptions.sql
psql -h localhost -U user -d paymentservice -f migrations/0007_usage.sql
```

2. **Setup Redis**
```bash
docker run -d --name redis -p 6379:6379 redis:6-alpine
```

3. **Configure Environment**
```bash
cp config.yaml.example config.yaml
# Edit config.yaml with your settings
```

4. **Run Application**
```bash
go run cmd/paymentservice/main.go
```

### Docker Deployment

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o payment-service cmd/paymentservice/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/payment-service .
COPY --from=builder /app/config.yaml .
CMD ["./payment-service"]
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
        - containerPort: 8080
        - containerPort: 8081
        env:
        - name: POSTGRES_DSN
          valueFrom:
            secretKeyRef:
              name: payment-service-secrets
              key: postgres-dsn
        - name: REDIS_ADDR
          value: "redis:6379"
        - name: STRIPE_SECRET
          valueFrom:
            secretKeyRef:
              name: payment-service-secrets
              key: stripe-secret
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8081
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8081
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_DSN` | PostgreSQL connection string | Required |
| `REDIS_ADDR` | Redis server address | `localhost:6379` |
| `STRIPE_SECRET` | Stripe secret key | Required |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook secret | Required |
| `LOG_LEVEL` | Logging level | `info` |
| `GRPC_ADDRESS` | gRPC server address | `:8080` |
| `HEALTH_PORT` | Health check port | `8081` |

---

## Summary

This Payment Service provides a comprehensive, production-ready solution for handling payments, subscriptions, and entitlements. The architecture follows clean architecture principles with clear separation of concerns, making it maintainable and testable.

Key architectural decisions:
- **gRPC for high-performance communication**
- **Clean architecture with clear layer separation**
- **Circuit breakers for resilience**
- **Comprehensive monitoring and metrics**
- **Secure webhook processing**
- **Redis-based caching and rate limiting**
- **PostgreSQL for reliable data persistence**

The service is designed to scale horizontally and handle high-volume payment processing with proper error handling, monitoring, and observability.

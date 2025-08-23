# Payment Service

A Go-based microservice for handling payment processing operations with plans and entitlements management, built with gRPC, PostgreSQL, and Redis.

## Features

- **gRPC API**: High-performance RPC service for payment operations
- **PostgreSQL**: Persistent storage with migrations
- **Redis**: Caching layer for improved performance
- **Structured Logging**: Using Zap logger with context-aware logging
- **Configuration Management**: Using Viper
- **Docker Support**: Containerized deployment
- **Health Checks**: Built-in health monitoring

## Project Structure

```
.
├── cmd/paymentservice/     # Application entry point
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── domain/            # Business logic and models
│   ├── log/               # Logging setup
│   ├── repository/        # Data access layer
│   ├── server/            # gRPC server implementation
│   ├── service/           # Business logic services
│   ├── cache/             # Redis cache implementation
│   └── events/            # Event publishing
├── proto/                 # Protocol buffer definitions
├── migrations/            # Database migrations
├── docker/                # Docker configuration
├── sqlc.yaml             # SQL code generation config
└── Makefile              # Build and deployment commands
```

## Prerequisites

- Go 1.21+
- Docker and Docker Compose
- PostgreSQL 16
- Redis 7

## Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd jia_family_app
   ```

2. **Start dependencies**
   ```bash
   cd docker
   docker-compose up postgres redis -d
   ```

3. **Run migrations**
   ```bash
   docker-compose --profile migrate run --rm migrate
   ```

4. **Build and run the service**
   ```bash
   cd ..
   make run
   ```

## Makefile Targets

- `make help` - Show available targets
- `make generate` - Generate code from proto and sqlc
- `make migrate-up` - Run database migrations
- `make migrate-down` - Rollback database migrations (1 step)
- `make migrate-create` - Create a new migration file
- `make migrate-force` - Force migration to specific version
- `make run` - Run the payment service
- `make lint` - Run linter and formatter

## Configuration

The service can be configured via environment variables or a `config.yaml` file. The configuration supports:

### Configuration Structure
- **AppName**: Application identifier
- **GRPC**: gRPC server address (default: `:8081`)
- **Postgres**: Database connection string and connection pool settings
- **Redis**: Cache server address, database number, and authentication
- **Auth**: Authentication configuration (TODO: integrate real provider)
- **Billing**: Billing provider settings (placeholder for Stripe integration)
- **Events**: Event streaming configuration (Kafka, etc.)
- **Log**: Logging level configuration

### Environment Variables
All configuration can be overridden via environment variables using dot notation:
```bash
export GRPC_ADDRESS=":50051"
export POSTGRES_DSN="host=db port=5432 user=paymentservice password=secret dbname=paymentservice"
export REDIS_ADDR="redis:6379"
export BILLING_PROVIDER="stripe"
export EVENTS_PROVIDER="kafka"
```

### Configuration File
See `config.yaml` for the complete configuration structure and defaults.

## Logging

The service uses structured logging with Zap. The logger provides:

### Context-Aware Logging
- **Request ID**: Automatically included in all log entries within a request
- **User ID**: Extracted from authentication context
- **Trace ID**: For distributed tracing support

### Usage Examples
```go
// Initialize logger
log.Init("info")

// Create context with request-scoped fields
ctx := log.WithRequestID(context.Background(), "req_123")
ctx = log.WithUserID(ctx, "user_456")

// Log with context
log.Info(ctx, "Processing payment", 
    zap.String("payment_id", "pay_123"),
    zap.Int64("amount", 1000))

// Direct logger access
log.L(ctx).Error("Payment failed", zap.Error(err))
```

### Log Levels
- `debug`: Detailed debugging information
- `info`: General information about service operation
- `warn`: Warning messages for potentially harmful situations
- `error`: Error messages for failures

## Database Schema

The service manages two main entities:

### Plans Table
- Subscription plans with feature codes, pricing, and billing cycles
- Supports family plans with user limits
- JSONB fields for usage limits and metadata
- TEXT[] array for feature codes

### Entitlements Table  
- User/family feature entitlements linked to plans
- Expiration tracking for time-limited features
- Status management (active, expired, cancelled)
- Foreign key relationship to plans with cascade updates

## API

The service exposes a gRPC API with the following operations:

- `CreatePayment` - Create a new payment
- `GetPayment` - Retrieve a payment by ID
- `UpdatePaymentStatus` - Update payment status
- `GetPaymentsByCustomer` - Get payments for a customer
- `ListPayments` - List payments with pagination

## Development

1. **Install development dependencies**
   ```bash
   go install github.com/kyleconroy/sqlc@latest
   go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   ```

2. **Generate code**
   ```bash
   make generate
   ```

3. **Run tests**
   ```bash
   go test ./...
   ```

## Docker

Build the Docker image:
```bash
docker build -f docker/Dockerfile -t paymentservice .
```

Run with Docker Compose:
```bash
cd docker
docker-compose up -d
```

## License

[Add your license here]

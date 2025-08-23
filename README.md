# Payment Service

A Go-based microservice for handling payment processing operations with plans and entitlements management, built with gRPC, PostgreSQL, and Redis.

## Features

- **gRPC API**: High-performance RPC service for payment operations
- **PostgreSQL**: Persistent storage with migrations
- **Redis**: Caching layer for improved performance
- **Structured Logging**: Using Zap logger
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

The service can be configured via environment variables or a `config.yaml` file. See `config.yaml` for available options.

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

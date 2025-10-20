# Payment Service Quick Reference Guide

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- PostgreSQL 13+
- Redis 6+
- Stripe account

### Local Setup
```bash
# 1. Clone and setup
git clone <repository>
cd jia_family_app

# 2. Start dependencies
docker-compose up -d postgres redis

# 3. Run migrations
psql -h localhost -U user -d paymentservice -f migrations/0006_subscriptions.sql
psql -h localhost -U user -d paymentservice -f migrations/0007_usage.sql

# 4. Configure
cp config.yaml.example config.yaml
# Edit config.yaml with your Stripe keys

# 5. Run
go run cmd/paymentservice/main.go
```

## 📁 Key Files & Their Purpose

| File | Purpose | When to Modify |
|------|--------|----------------|
| `api/payment/v1/payment_service.proto` | gRPC API definition | Adding new endpoints |
| `internal/payment/transport/grpc.go` | gRPC handlers | Request/response handling |
| `internal/payment/usecase/` | Business logic | Core functionality |
| `internal/payment/repo/postgres/store.go` | Database operations | Data access |
| `internal/payment/webhook/` | Webhook processing | Stripe integration |
| `internal/shared/metrics/` | Monitoring | Observability |
| `config.yaml` | Configuration | Environment settings |

## 🔄 Common Workflows

### Adding a New API Endpoint

1. **Define in Proto** (`api/payment/v1/payment_service.proto`)
```protobuf
rpc NewMethod(NewMethodRequest) returns (NewMethodResponse);
```

2. **Regenerate Code**
```bash
protoc --go_out=. --go-grpc_out=. api/payment/v1/payment_service.proto
```

3. **Implement Handler** (`internal/payment/transport/grpc.go`)
```go
func (s *PaymentService) NewMethod(ctx context.Context, req *paymentv1.NewMethodRequest) (*paymentv1.NewMethodResponse, error) {
    // Implementation
}
```

4. **Add Business Logic** (`internal/payment/usecase/`)
5. **Add Repository Methods** (`internal/payment/repo/`)
6. **Add SQL Queries** (`internal/payment/repo/postgres/queries/`)

### Adding a New Webhook Event

1. **Update Parser** (`internal/payment/webhook/parser.go`)
```go
case stripe.NewEventType:
    // Parse new event type
```

2. **Update Validator** (`internal/payment/webhook/validator.go`)
3. **Add Business Logic** (`internal/payment/usecase/`)

### Adding Database Tables

1. **Create Migration** (`migrations/`)
```sql
-- migrations/0008_new_table.sql
CREATE TABLE new_table (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- columns
);
```

2. **Add SQL Queries** (`internal/payment/repo/postgres/queries/`)
3. **Generate Code**
```bash
sqlc generate
```

4. **Update Repository** (`internal/payment/repo/postgres/store.go`)

## 🛠 Development Commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Generate protobuf
protoc --go_out=. --go-grpc_out=. api/payment/v1/payment_service.proto

# Generate SQL code
sqlc generate

# Run linter
golangci-lint run

# Format code
go fmt ./...

# Tidy dependencies
go mod tidy
```

## 🔍 Debugging

### Check Service Health
```bash
curl http://localhost:8081/health
curl http://localhost:8081/health/ready
curl http://localhost:8081/metrics
```

### View Logs
```bash
# Application logs
go run cmd/paymentservice/main.go

# Database logs
docker logs postgres

# Redis logs
docker logs redis
```

### Test gRPC Calls
```bash
# Using grpcurl
grpcurl -plaintext localhost:8080 list
grpcurl -plaintext localhost:8080 payment.v1.PaymentService/CheckEntitlement -d '{"user_id":"test", "feature_code":"premium"}'
```

## 🚨 Common Issues & Solutions

### Issue: "could not import github.com/jia-app/paymentservice/api/payment/v1"
**Solution**: Regenerate protobuf files
```bash
protoc --go_out=. --go-grpc_out=. api/payment/v1/payment_service.proto
```

### Issue: "missing method GetBySubscriptionID"
**Solution**: Add missing repository methods
```bash
# Add SQL query, then:
sqlc generate
```

### Issue: "circuit breaker state undefined"
**Solution**: Check circuit breaker configuration
```go
// In your service initialization
circuitBreaker := circuitbreaker.GetOrCreateGlobal("stripe", circuitbreaker.StripeConfig)
```

### Issue: "rate limit exceeded"
**Solution**: Check Redis connection and rate limit configuration
```bash
# Check Redis
redis-cli ping

# Check rate limit config in config.yaml
```

## 📊 Monitoring & Observability

### Key Metrics to Watch
- `payment_success_total` / `payment_failed_total`
- `entitlement_cache_hits_total` / `entitlement_cache_misses_total`
- `db_query_duration_seconds`
- `circuit_breaker_state`

### Health Check Endpoints
- `/health` - Full health check
- `/health/ready` - Kubernetes readiness
- `/health/live` - Kubernetes liveness
- `/metrics` - Prometheus metrics

### Log Levels
- `debug` - Detailed debugging info
- `info` - General information
- `warn` - Warning messages
- `error` - Error messages

## 🔐 Security Considerations

### Webhook Security
- Always validate Stripe webhook signatures
- Check timestamp to prevent replay attacks
- Use HTTPS for webhook endpoints

### Database Security
- Use connection pooling
- Implement proper access controls
- Encrypt sensitive data

### API Security
- Implement authentication interceptors
- Use rate limiting
- Validate all input data

## 🚀 Production Deployment

### Environment Variables
```bash
export POSTGRES_DSN="postgres://user:pass@host:5432/db"
export REDIS_ADDR="redis:6379"
export STRIPE_SECRET="sk_live_..."
export STRIPE_WEBHOOK_SECRET="whsec_..."
export LOG_LEVEL="info"
```

### Docker Build
```bash
docker build -t payment-service .
docker run -p 8080:8080 -p 8081:8081 payment-service
```

### Kubernetes
```bash
kubectl apply -f k8s/
kubectl get pods
kubectl logs -f deployment/payment-service
```

## 📚 Additional Resources

- [gRPC Documentation](https://grpc.io/docs/)
- [Stripe Webhooks](https://stripe.com/docs/webhooks)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)
- [Prometheus Metrics](https://prometheus.io/docs/)

## 🤝 Contributing

1. Follow Go best practices
2. Add tests for new functionality
3. Update documentation
4. Run linter before committing
5. Use conventional commit messages

---

**Need Help?** Check the main architecture guide or create an issue in the repository.

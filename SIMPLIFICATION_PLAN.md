# Payment Service Simplification Plan

## Current State Analysis

### Over-Engineered Components
1. **Service Mesh** (Envoy + Spiffe + Discovery) - NOT NEEDED
2. **Service Manager** - Only managing 1 service, overkill
3. **Circuit Breakers** - Premature optimization
4. **Dunning Management** - Not implemented, dead code
5. **Usage Tracker** - Not implemented, dead code
6. **Complex Config** - 238 lines for simple service

### What Works Well
- ✅ Core payment processing
- ✅ Entitlement checking with caching
- ✅ Checkout session creation
- ✅ Webhook processing
- ✅ Database layer
- ✅ Basic Stripe integration

## Simplification Steps

### Phase 1: Remove Unused Complexity (Priority: HIGH)

#### 1.1 Remove Service Mesh Integration
**Files to Remove:**
- `internal/shared/discovery/` (entire directory)
- `internal/shared/services/service_manager.go`
- `internal/shared/services/contact_service_client.go`
- `internal/shared/circuitbreaker/` (entire directory)

**Files to Simplify:**
- `internal/app/app.go` - Remove service manager initialization
- `internal/shared/config/config.go` - Remove service mesh configs
- `config.yaml` - Remove service mesh sections

**Benefit:** Removes ~800 lines of unused code

#### 1.2 Remove Dead Features
**Files to Remove:**
- `internal/payment/usecase/dunning_manager.go`
- `internal/payment/usecase/dunning_scheduler.go`
- `internal/payment/usecase/usage_tracker.go`
- `internal/payment/memory_store.go` (if not used)

**Benefit:** Removes ~400 lines of dead code

#### 1.3 Simplify Configuration
**Keep Only:**
```yaml
app_name: payment-service
grpc:
  address: ":8081"
  enable_reflection: true

postgres:
  dsn: "postgresql://..."
  max_conns: 10

redis:
  addr: "localhost:6379"
  db: 0
  password: ""

auth:
  public_key_pem: ""

billing:
  provider: stripe
  stripe_secret: ""
  stripe_webhook_secret: ""

log:
  level: info
```

**Remove:**
- Service mesh config
- mTLS config
- External services config
- Circuit breaker config
- API gateway config

**Benefit:** Config goes from 238 lines to ~50 lines

### Phase 2: Simplify Core Structure (Priority: MEDIUM)

#### 2.1 Simplify App Initialization
**Current:** Complex initialization with service mesh, service manager, circuit breakers

**Simplified:**
```go
type App struct {
    config      *config.Config
    logger      *zap.Logger
    dbPool      *pgxpool.Pool
    redisClient *redis.Client
    grpcServer  *server.GRPCServer
}

func New(cfg *config.Config) (*App, error) {
    // Initialize logger
    if err := log.Init(cfg.Log.Level); err != nil {
        return nil, err
    }
    logger := log.L(context.Background())

    // Initialize database
    dbPool, err := initializeDatabase(cfg)
    if err != nil {
        return nil, err
    }

    // Initialize Redis (optional)
    redisClient, err := initializeRedis(cfg)
    if err != nil {
        logger.Warn("Redis failed, continuing without cache")
        redisClient = nil
    }

    // Initialize gRPC server with payment service
    grpcServer := server.NewGRPCServer(cfg, dbPool, redisClient)

    return &App{
        config:      cfg,
        logger:      logger,
        dbPool:      dbPool,
        redisClient: redisClient,
        grpcServer:  grpcServer,
    }, nil
}
```

**Benefit:** App initialization goes from 206 lines to ~80 lines

#### 2.2 Simplify Payment Service Transport
**Current:** Complex with many dependencies

**Simplified:** Keep only essential dependencies
```go
type PaymentService struct {
    paymentv1.UnimplementedPaymentServiceServer
    config               *config.Config
    paymentUseCase       *usecase.PaymentUseCase
    entitlementUseCase   *usecase.EntitlementUseCase
    checkoutUseCase      *usecase.CheckoutUseCase
    pricingZoneUseCase   *usecase.PricingZoneUseCase
    cache                *cache.Cache
    billingProvider      billing.Provider
    webhookValidator     *webhook.Validator
    webhookParser        *webhook.Parser
}
```

**Remove:**
- BulkEntitlementUseCase (can be added later if needed)
- EntitlementPublisher (use simple events)
- MetricsCollector (use basic logging for now)

### Phase 3: Clean Up Dependencies (Priority: LOW)

#### 3.1 Simplify Stripe Integration
**Current:** Full circuit breaker wrapper

**Simplified:** Simple retry logic
```go
func (a *Adapter) CreateCheckoutSession(ctx context.Context, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error) {
    stripe.Key = a.secretKey
    
    // Simple retry logic (3 attempts)
    var err error
    for i := 0; i < 3; i++ {
        session, err := session.New(params)
        if err == nil {
            return convertToResponse(session), nil
        }
        time.Sleep(time.Duration(i+1) * time.Second)
    }
    
    return nil, fmt.Errorf("failed after 3 attempts: %w", err)
}
```

#### 3.2 Simplify Event Publishing
**Current:** Complex event bus setup

**Simplified:** Direct function calls or simple pub/sub
```go
// Simple event publishing
func (uc *EntitlementUseCase) CreateEntitlement(ctx context.Context, ...) (*domain.Entitlement, error) {
    // Create entitlement
    entitlement, err := uc.entitlementRepo.Insert(ctx, entitlement)
    if err != nil {
        return nil, err
    }
    
    // Simple event notification (can be enhanced later)
    log.Info(ctx, "Entitlement created", 
        zap.String("user_id", entitlement.UserID),
        zap.String("feature", entitlement.FeatureCode))
    
    return &entitlement, nil
}
```

## Implementation Priority

### Must Do (Week 1)
1. ✅ Remove service mesh code
2. ✅ Remove service manager
3. ✅ Remove circuit breakers
4. ✅ Simplify configuration

### Should Do (Week 2)
5. ✅ Remove dead features (dunning, usage tracker)
6. ✅ Simplify app initialization
7. ✅ Simplify payment service transport

### Nice to Have (Week 3)
8. ✅ Simplify Stripe integration
9. ✅ Simplify event publishing
10. ✅ Add comprehensive tests

## Expected Results

### Code Reduction
- **Before:** ~15,000 lines of code
- **After:** ~8,000 lines of code
- **Reduction:** ~47% less code

### Complexity Reduction
- **Before:** 12 major components
- **After:** 6 essential components
- **Benefit:** Much easier to understand and maintain

### Performance
- **Startup Time:** Faster (no service discovery)
- **Memory:** Lower (fewer components)
- **Maintainability:** Much better

## Migration Path

1. **Create feature branch:** `simplify-payment-service`
2. **Remove unused code** (Phase 1)
3. **Simplify core** (Phase 2)
4. **Test thoroughly**
5. **Merge to main**

## Testing Strategy

After simplification:
1. Run existing tests
2. Test payment flow end-to-end
3. Test entitlement checking
4. Test webhook processing
5. Load test with basic scenarios

## Rollback Plan

If issues arise:
1. Keep old code in separate branch
2. Use feature flags if needed
3. Gradual rollout

## Success Metrics

- ✅ Code reduced by 40%+
- ✅ No functionality lost
- ✅ Tests still pass
- ✅ Startup time improved
- ✅ Easier for new developers to understand


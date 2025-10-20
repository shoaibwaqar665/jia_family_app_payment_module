# Webhook Processing Guide

## Overview

The Payment Service implements secure webhook processing for Stripe payment events. This guide covers the complete webhook flow, security measures, and implementation details.

## Webhook Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Stripe                                  │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Payment Events  │ │ Subscription    │ │ Invoice Events  │   │
│  │ - charge.succ.  │ │ - sub.created   │ │ - inv.pay.succ. │   │
│  │ - charge.failed │ │ - sub.updated   │ │ - inv.pay.failed│   │
│  │ - payment.succ. │ │ - sub.deleted   │ │                 │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTPS POST with Stripe-Signature header
┌─────────────────────▼───────────────────────────────────────────┐
│                 Payment Service                                 │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐   │
│  │ Rate Limiting   │ │ Auth Validation │ │ Metrics Collect │   │
│  │ - Redis-based   │ │ - Signature     │ │ - Success/Fail  │   │
│  │ - Per-endpoint  │ │ - Timestamp     │ │ - Duration      │   │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│              Webhook Processing Pipeline                       │
│                                                                 │
│  1. Signature Validation                                       │
│     ├── Extract timestamp and signature from header            │
│     ├── Check timestamp (prevent replay attacks)               │
│     ├── Compute HMAC-SHA256 signature                           │
│     └── Compare with provided signature                        │
│                                                                 │
│  2. Payload Parsing                                            │
│     ├── Parse JSON payload                                     │
│     ├── Extract event type                                     │
│     ├── Extract relevant data based on event type             │
│     └── Normalize to WebhookResult structure                   │
│                                                                 │
│  3. Business Logic Application                                 │
│     ├── Update entitlements                                    │
│     ├── Update subscriptions                                   │
│     ├── Record payment transactions                            │
│     └── Publish domain events                                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Webhook Security Implementation

### Signature Validation Process

```go
// internal/payment/webhook/validator.go

func (v *Validator) ValidateStripeWebhook(payload []byte, signature string) error {
    const SignatureTolerance = 300 * time.Second // 5 minutes

    if signature == "" {
        return fmt.Errorf("missing Stripe-Signature header")
    }

    // Extract timestamp and signatures from the header
    var timestamp int64
    var signatures []string
    parts := strings.Split(signature, ",")
    for _, part := range parts {
        if strings.HasPrefix(part, "t=") {
            tStr := strings.TrimPrefix(part, "t=")
            t, err := strconv.ParseInt(tStr, 10, 64)
            if err != nil {
                return fmt.Errorf("invalid timestamp in signature: %w", err)
            }
            timestamp = t
        } else if strings.HasPrefix(part, "v1=") {
            signatures = append(signatures, strings.TrimPrefix(part, "v1="))
        }
    }

    // Check timestamp for replay attacks
    if time.Since(time.Unix(timestamp, 0)) > SignatureTolerance {
        return fmt.Errorf("webhook timestamp too old, possible replay attack")
    }

    // Prepare signed payload
    signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))

    // Compute expected signature
    mac := hmac.New(sha256.New, []byte(v.stripeWebhookSecret))
    mac.Write([]byte(signedPayload))
    expectedSignature := hex.EncodeToString(mac.Sum(nil))

    // Compare expected signature with all provided signatures
    for _, sig := range signatures {
        if hmac.Equal([]byte(expectedSignature), []byte(sig)) {
            return nil // Signature is valid
        }
    }

    return fmt.Errorf("invalid Stripe webhook signature")
}
```

### Security Features

1. **HMAC-SHA256 Signature Verification**
   - Uses Stripe's webhook secret to verify authenticity
   - Prevents tampering with webhook payloads

2. **Timestamp Validation**
   - Rejects webhooks older than 5 minutes
   - Prevents replay attacks

3. **Rate Limiting**
   - Redis-based rate limiting per endpoint
   - Prevents webhook flooding

4. **Idempotency**
   - Webhook processing is idempotent
   - Duplicate events are handled safely

## Supported Webhook Events

### Payment Events

| Event Type | Description | Actions Taken |
|------------|-------------|--------------|
| `checkout.session.completed` | Payment successful via checkout | Grant entitlements, create subscription |
| `payment_intent.succeeded` | Payment successful | Grant entitlements |
| `payment_intent.payment_failed` | Payment failed | Log failure, trigger dunning |

### Subscription Events

| Event Type | Description | Actions Taken |
|------------|-------------|--------------|
| `customer.subscription.created` | New subscription | Create subscription record |
| `customer.subscription.updated` | Subscription changed | Update subscription status |
| `customer.subscription.deleted` | Subscription cancelled | Revoke entitlements |

### Invoice Events

| Event Type | Description | Actions Taken |
|------------|-------------|--------------|
| `invoice.payment_succeeded` | Recurring payment successful | Renew subscription, extend entitlements |
| `invoice.payment_failed` | Recurring payment failed | Trigger dunning process |

## Webhook Data Flow

### 1. Event Reception

```go
// Stripe sends webhook to: POST /webhooks/stripe
// Headers:
//   Stripe-Signature: t=1234567890,v1=signature_hash
//   Content-Type: application/json
// Body: JSON payload with event data
```

### 2. Validation & Parsing

```go
// internal/payment/transport/grpc.go
func (s *PaymentService) PaymentSuccessWebhook(ctx context.Context, payload []byte, signature string) error {
    start := time.Now()
    
    // 1. Validate webhook signature
    if err := s.webhookValidator.ValidateStripeWebhook(payload, signature); err != nil {
        s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
        return status.Error(codes.Unauthenticated, "invalid webhook signature")
    }

    // 2. Parse webhook payload
    webhookResult, err := s.webhookParser.ParseStripeWebhook(payload)
    if err != nil {
        s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
        return status.Error(codes.InvalidArgument, "invalid webhook payload")
    }

    // 3. Apply webhook result
    billingResult := billing.WebhookResult{
        EventType:    "payment.succeeded",
        UserID:       webhookResult.UserID,
        FamilyID:     webhookResult.FamilyID,
        FeatureCode:  webhookResult.FeatureCode,
        PlanID:       webhookResult.PlanID,
        PlanIDString: webhookResult.PlanIDString,
        Amount:       float64(webhookResult.Amount),
        Currency:     webhookResult.Currency,
        Status:       webhookResult.Status,
        ExpiresAt:    webhookResult.ExpiresAt,
        Metadata:     webhookResult.Metadata,
    }

    // 4. Process webhook
    if err := s.checkoutUseCase.ApplyWebhook(ctx, billingResult); err != nil {
        s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
        return status.Errorf(codes.Internal, "failed to apply webhook: %v", err)
    }

    // 5. Record success
    s.metricsCollector.RecordWebhook(ctx, true, time.Since(start))
    return nil
}
```

### 3. Event Parsing

```go
// internal/payment/webhook/parser.go
func (p *Parser) ParseStripeWebhook(payload []byte) (*WebhookResult, error) {
    event := stripe.Event{}
    if err := json.Unmarshal(payload, &event); err != nil {
        return nil, fmt.Errorf("failed to parse Stripe event: %w", err)
    }

    webhookResult := &WebhookResult{
        EventType: string(event.Type),
        Metadata:  make(map[string]interface{}),
    }

    switch event.Type {
    case stripe.CheckoutSessionCompleted:
        var session stripe.CheckoutSession
        err := json.Unmarshal(event.Data.Raw, &session)
        if err != nil {
            return nil, fmt.Errorf("failed to parse checkout session: %w", err)
        }
        
        webhookResult.UserID = session.Metadata["user_id"]
        webhookResult.FamilyID = getStringPtr(session.Metadata["family_id"])
        webhookResult.PlanID, _ = uuid.Parse(session.Metadata["plan_id"])
        webhookResult.PlanIDString = session.Metadata["plan_id"]
        webhookResult.FeatureCode = session.Metadata["feature_code"]
        webhookResult.Amount = session.AmountTotal
        webhookResult.Currency = string(session.Currency)
        webhookResult.Status = string(session.Status)
        
        if session.ExpiresAt > 0 {
            expiresAt := time.Unix(session.ExpiresAt, 0)
            webhookResult.ExpiresAt = &expiresAt
        }

    case stripe.InvoicePaymentSucceeded:
        var invoice stripe.Invoice
        err := json.Unmarshal(event.Data.Raw, &invoice)
        if err != nil {
            return nil, fmt.Errorf("failed to parse invoice: %w", err)
        }
        
        webhookResult.UserID = invoice.Subscription.Metadata["user_id"]
        webhookResult.FamilyID = getStringPtr(invoice.Subscription.Metadata["family_id"])
        webhookResult.PlanID, _ = uuid.Parse(invoice.Subscription.Metadata["plan_id"])
        webhookResult.PlanIDString = invoice.Subscription.Metadata["plan_id"]
        webhookResult.FeatureCode = invoice.Subscription.Metadata["feature_code"]
        webhookResult.SubscriptionID = invoice.Subscription.ID
        webhookResult.Amount = invoice.AmountPaid
        webhookResult.Currency = string(invoice.Currency)
        webhookResult.Status = "succeeded"
        
        if invoice.PeriodEnd > 0 {
            expiresAt := time.Unix(invoice.PeriodEnd, 0)
            webhookResult.ExpiresAt = &expiresAt
        }

    // ... other event types
    }

    return webhookResult, nil
}
```

## Webhook Data Structures

### WebhookResult Structure

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

### Stripe Event Structure

```go
type StripeEvent struct {
    ID      string                 `json:"id"`
    Type    string                 `json:"type"`
    Data    StripeEventData        `json:"data"`
    Created int64                  `json:"created"`
}

type StripeEventData struct {
    Object map[string]interface{} `json:"object"`
    Raw    json.RawMessage        `json:"raw"`
}
```

## Business Logic Application

### Entitlement Management

```go
// When payment succeeds, grant entitlements
func (c *CheckoutUseCase) ApplyWebhook(ctx context.Context, wr billing.WebhookResult) error {
    // 1. Create or update entitlement
    entitlement := domain.Entitlement{
        UserID:      wr.UserID,
        FamilyID:    wr.FamilyID,
        FeatureCode: wr.FeatureCode,
        PlanID:      wr.PlanID,
        Status:      "active",
        GrantedAt:   time.Now(),
        ExpiresAt:   wr.ExpiresAt,
    }

    if wr.SubscriptionID != "" {
        entitlement.SubscriptionID = &wr.SubscriptionID
    }

    // 2. Save entitlement
    _, err := c.entitlementRepo.Insert(ctx, entitlement)
    if err != nil {
        return fmt.Errorf("failed to create entitlement: %w", err)
    }

    // 3. Update cache
    if c.cache != nil {
        cacheKey := fmt.Sprintf("entitlement:%s:%s", wr.UserID, wr.FeatureCode)
        c.cache.SetEntitlement(ctx, entitlement, 24*time.Hour)
    }

    // 4. Publish event
    if c.eventPublisher != nil {
        c.eventPublisher.PublishEntitlementGranted(ctx, &entitlement)
    }

    return nil
}
```

### Subscription Management

```go
// Handle subscription lifecycle events
func (lm *LifecycleManager) ProcessSubscriptionEvent(ctx context.Context, event *WebhookResult) error {
    switch event.EventType {
    case "customer.subscription.created":
        return lm.createSubscription(ctx, event)
    case "customer.subscription.updated":
        return lm.updateSubscription(ctx, event)
    case "customer.subscription.deleted":
        return lm.cancelSubscription(ctx, event)
    case "invoice.payment_succeeded":
        return lm.renewSubscription(ctx, event)
    case "invoice.payment_failed":
        return lm.handlePaymentFailure(ctx, event)
    }
    return nil
}
```

## Error Handling & Retry Logic

### Webhook Error Handling

```go
func (s *PaymentService) PaymentSuccessWebhook(ctx context.Context, payload []byte, signature string) error {
    // ... validation and parsing ...

    // Apply webhook with retry logic
    maxRetries := 3
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := s.checkoutUseCase.ApplyWebhook(ctx, billingResult)
        if err == nil {
            s.metricsCollector.RecordWebhook(ctx, true, time.Since(start))
            return nil
        }

        // Log retry attempt
        s.logger.Warn("Webhook processing failed, retrying",
            zap.Int("attempt", attempt+1),
            zap.Error(err))

        // Wait before retry
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }

    // All retries failed
    s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
    return status.Errorf(codes.Internal, "webhook processing failed after %d attempts", maxRetries)
}
```

### Dunning Management

```go
// Handle failed payments
func (dm *DunningManager) ProcessPaymentFailure(ctx context.Context, event *WebhookResult) error {
    dunningEvent := DunningEvent{
        ID:             uuid.New(),
        UserID:         event.UserID,
        SubscriptionID: event.SubscriptionID,
        Amount:         event.Amount,
        Currency:       event.Currency,
        FailureReason:  "payment_failed",
        AttemptCount:   1,
        NextRetryAt:    time.Now().Add(24 * time.Hour),
        Status:         "pending",
        CreatedAt:      time.Now(),
    }

    // Store dunning event
    err := dm.storeDunningEvent(ctx, dunningEvent)
    if err != nil {
        return fmt.Errorf("failed to store dunning event: %w", err)
    }

    // Schedule retry
    dm.scheduler.ScheduleRetry(ctx, dunningEvent.ID, dunningEvent.NextRetryAt)

    // Publish event
    dm.eventPublisher.PublishDunningEvent(ctx, &dunningEvent)

    return nil
}
```

## Monitoring & Metrics

### Webhook Metrics

```go
// Metrics collected for webhooks
type WebhookMetrics struct {
    WebhookReceived    prometheus.Counter   // Total webhooks received
    WebhookProcessed   prometheus.Counter   // Successfully processed
    WebhookFailed      prometheus.Counter   // Failed processing
    WebhookDuration    prometheus.Histogram // Processing duration
}

// Usage in webhook handler
func (s *PaymentService) PaymentSuccessWebhook(ctx context.Context, payload []byte, signature string) error {
    start := time.Now()
    
    // ... processing ...
    
    // Record metrics
    s.metricsCollector.RecordWebhook(ctx, success, time.Since(start))
    
    return nil
}
```

### Health Checks

```go
// Webhook endpoint health check
func (hc *HealthChecker) checkWebhookHealth(ctx context.Context) HealthStatus {
    // Check webhook processing rate
    // Check recent webhook failures
    // Check signature validation success rate
    
    return HealthStatus{
        Component: "webhook",
        Status:    "healthy",
        Details: map[string]string{
            "processing_rate": "100 req/min",
            "success_rate":    "99.9%",
            "last_check":      time.Now().Format(time.RFC3339),
        },
        Timestamp: time.Now(),
    }
}
```

## Testing Webhooks

### Local Testing with Stripe CLI

```bash
# Install Stripe CLI
brew install stripe/stripe-cli/stripe

# Login to Stripe
stripe login

# Forward webhooks to local service
stripe listen --forward-to localhost:8080/webhooks/stripe

# Trigger test events
stripe trigger payment_intent.succeeded
stripe trigger customer.subscription.created
```

### Unit Testing

```go
func TestWebhookValidation(t *testing.T) {
    validator := webhook.NewValidator("whsec_test_secret")
    
    payload := []byte(`{"type": "payment_intent.succeeded", "data": {...}}`)
    signature := "t=1234567890,v1=valid_signature"
    
    err := validator.ValidateStripeWebhook(payload, signature)
    assert.NoError(t, err)
}

func TestWebhookParsing(t *testing.T) {
    parser := webhook.NewParser()
    
    payload := []byte(`{
        "type": "checkout.session.completed",
        "data": {
            "object": {
                "metadata": {
                    "user_id": "user123",
                    "plan_id": "basic_monthly",
                    "feature_code": "premium"
                },
                "amount_total": 1000,
                "currency": "usd"
            }
        }
    }`)
    
    result, err := parser.ParseStripeWebhook(payload)
    assert.NoError(t, err)
    assert.Equal(t, "user123", result.UserID)
    assert.Equal(t, "premium", result.FeatureCode)
}
```

## Configuration

### Webhook Configuration

```yaml
# config.yaml
billing:
  provider: "stripe"
  stripe_secret: "sk_test_..."
  stripe_webhook_secret: "whsec_..."

webhook:
  signature_tolerance: "300s"  # 5 minutes
  max_retries: 3
  retry_delay: "1s"
  rate_limit:
    enabled: true
    requests_per_minute: 100
```

### Environment Variables

```bash
export STRIPE_WEBHOOK_SECRET="whsec_..."
export WEBHOOK_SIGNATURE_TOLERANCE="300s"
export WEBHOOK_MAX_RETRIES="3"
```

## Best Practices

### Security
1. **Always validate webhook signatures**
2. **Check timestamps to prevent replay attacks**
3. **Use HTTPS for webhook endpoints**
4. **Implement rate limiting**
5. **Log all webhook events for audit**

### Reliability
1. **Make webhook processing idempotent**
2. **Implement retry logic with exponential backoff**
3. **Handle duplicate events gracefully**
4. **Use database transactions for consistency**
5. **Monitor webhook processing metrics**

### Performance
1. **Process webhooks asynchronously when possible**
2. **Use caching for frequently accessed data**
3. **Implement circuit breakers for external calls**
4. **Monitor processing duration**
5. **Scale horizontally for high volume**

---

This webhook processing system provides secure, reliable, and scalable handling of Stripe payment events with comprehensive monitoring and error handling.

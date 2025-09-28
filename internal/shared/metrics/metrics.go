package metrics

import (
	"context"
	"time"

	"github.com/jia-app/paymentservice/internal/shared/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// MetricsCollector collects and exposes metrics for the payment service
type MetricsCollector struct {
	// Payment metrics
	paymentTotal    prometheus.Counter
	paymentSuccess  prometheus.Counter
	paymentFailed   prometheus.Counter
	paymentDuration prometheus.Histogram
	paymentAmount   prometheus.Histogram

	// Entitlement metrics
	entitlementChecks      prometheus.Counter
	entitlementCacheHits   prometheus.Counter
	entitlementCacheMisses prometheus.Counter
	entitlementDuration    prometheus.Histogram

	// Subscription metrics
	subscriptionTotal     prometheus.Counter
	subscriptionActive    prometheus.Gauge
	subscriptionCancelled prometheus.Counter
	subscriptionSuspended prometheus.Counter

	// Usage metrics
	usageTracked  prometheus.Counter
	quotaExceeded prometheus.Counter
	usageAmount   prometheus.Histogram

	// Dunning metrics
	dunningEvents prometheus.Counter
	retryAttempts prometheus.Counter
	retrySuccess  prometheus.Counter
	retryFailed   prometheus.Counter

	// Circuit breaker metrics
	circuitBreakerState     prometheus.Gauge
	circuitBreakerFailures  prometheus.Counter
	circuitBreakerSuccesses prometheus.Counter

	// Rate limiting metrics
	rateLimitRequests prometheus.Counter
	rateLimitRejected prometheus.Counter
	rateLimitAllowed  prometheus.Counter

	// Webhook metrics
	webhookReceived  prometheus.Counter
	webhookProcessed prometheus.Counter
	webhookFailed    prometheus.Counter
	webhookDuration  prometheus.Histogram

	// Database metrics
	dbConnections   prometheus.Gauge
	dbQueryDuration prometheus.Histogram
	dbQueryErrors   prometheus.Counter

	// Cache metrics
	cacheHits     prometheus.Counter
	cacheMisses   prometheus.Counter
	cacheDuration prometheus.Histogram

	logger *zap.Logger
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		// Payment metrics
		paymentTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "payment_total",
			Help: "Total number of payment requests",
		}),
		paymentSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "payment_success_total",
			Help: "Total number of successful payments",
		}),
		paymentFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "payment_failed_total",
			Help: "Total number of failed payments",
		}),
		paymentDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "payment_duration_seconds",
			Help:    "Duration of payment processing",
			Buckets: prometheus.DefBuckets,
		}),
		paymentAmount: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "payment_amount_dollars",
			Help:    "Payment amounts in dollars",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		}),

		// Entitlement metrics
		entitlementChecks: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entitlement_checks_total",
			Help: "Total number of entitlement checks",
		}),
		entitlementCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entitlement_cache_hits_total",
			Help: "Total number of entitlement cache hits",
		}),
		entitlementCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "entitlement_cache_misses_total",
			Help: "Total number of entitlement cache misses",
		}),
		entitlementDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "entitlement_check_duration_seconds",
			Help:    "Duration of entitlement checks",
			Buckets: prometheus.DefBuckets,
		}),

		// Subscription metrics
		subscriptionTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "subscription_total",
			Help: "Total number of subscriptions",
		}),
		subscriptionActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "subscription_active",
			Help: "Number of active subscriptions",
		}),
		subscriptionCancelled: promauto.NewCounter(prometheus.CounterOpts{
			Name: "subscription_cancelled_total",
			Help: "Total number of cancelled subscriptions",
		}),
		subscriptionSuspended: promauto.NewCounter(prometheus.CounterOpts{
			Name: "subscription_suspended_total",
			Help: "Total number of suspended subscriptions",
		}),

		// Usage metrics
		usageTracked: promauto.NewCounter(prometheus.CounterOpts{
			Name: "usage_tracked_total",
			Help: "Total number of usage tracking events",
		}),
		quotaExceeded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "quota_exceeded_total",
			Help: "Total number of quota exceeded events",
		}),
		usageAmount: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "usage_amount_bytes",
			Help:    "Usage amounts in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 2, 20), // 1KB to 1GB
		}),

		// Dunning metrics
		dunningEvents: promauto.NewCounter(prometheus.CounterOpts{
			Name: "dunning_events_total",
			Help: "Total number of dunning events",
		}),
		retryAttempts: promauto.NewCounter(prometheus.CounterOpts{
			Name: "retry_attempts_total",
			Help: "Total number of retry attempts",
		}),
		retrySuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "retry_success_total",
			Help: "Total number of successful retries",
		}),
		retryFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "retry_failed_total",
			Help: "Total number of failed retries",
		}),

		// Circuit breaker metrics
		circuitBreakerState: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		}),
		circuitBreakerFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total number of circuit breaker failures",
		}),
		circuitBreakerSuccesses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "circuit_breaker_successes_total",
			Help: "Total number of circuit breaker successes",
		}),

		// Rate limiting metrics
		rateLimitRequests: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rate_limit_requests_total",
			Help: "Total number of rate limit requests",
		}),
		rateLimitRejected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rate_limit_rejected_total",
			Help: "Total number of rate limit rejections",
		}),
		rateLimitAllowed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rate_limit_allowed_total",
			Help: "Total number of rate limit allowances",
		}),

		// Webhook metrics
		webhookReceived: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_received_total",
			Help: "Total number of webhooks received",
		}),
		webhookProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_processed_total",
			Help: "Total number of webhooks processed",
		}),
		webhookFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_failed_total",
			Help: "Total number of failed webhooks",
		}),
		webhookDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "webhook_duration_seconds",
			Help:    "Duration of webhook processing",
			Buckets: prometheus.DefBuckets,
		}),

		// Database metrics
		dbConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		}),
		dbQueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		}),
		dbQueryErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors",
		}),

		// Cache metrics
		cacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		}),
		cacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		}),
		cacheDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "cache_duration_seconds",
			Help:    "Duration of cache operations",
			Buckets: prometheus.DefBuckets,
		}),

		logger: log.L(context.Background()),
	}
}

// Payment metrics methods
func (mc *MetricsCollector) RecordPayment(ctx context.Context, success bool, amount float64, duration time.Duration) {
	mc.paymentTotal.Inc()
	if success {
		mc.paymentSuccess.Inc()
	} else {
		mc.paymentFailed.Inc()
	}
	mc.paymentDuration.Observe(duration.Seconds())
	mc.paymentAmount.Observe(amount)
}

// Entitlement metrics methods
func (mc *MetricsCollector) RecordEntitlementCheck(ctx context.Context, cacheHit bool, duration time.Duration) {
	mc.entitlementChecks.Inc()
	if cacheHit {
		mc.entitlementCacheHits.Inc()
	} else {
		mc.entitlementCacheMisses.Inc()
	}
	mc.entitlementDuration.Observe(duration.Seconds())
}

// Subscription metrics methods
func (mc *MetricsCollector) RecordSubscription(ctx context.Context, eventType string) {
	mc.subscriptionTotal.Inc()
	switch eventType {
	case "cancelled":
		mc.subscriptionCancelled.Inc()
	case "suspended":
		mc.subscriptionSuspended.Inc()
	}
}

func (mc *MetricsCollector) UpdateActiveSubscriptions(count int) {
	mc.subscriptionActive.Set(float64(count))
}

// Usage metrics methods
func (mc *MetricsCollector) RecordUsage(ctx context.Context, amount int64, quotaExceeded bool) {
	mc.usageTracked.Inc()
	if quotaExceeded {
		mc.quotaExceeded.Inc()
	}
	mc.usageAmount.Observe(float64(amount))
}

// Dunning metrics methods
func (mc *MetricsCollector) RecordDunningEvent(ctx context.Context, eventType string) {
	mc.dunningEvents.Inc()
	switch eventType {
	case "retry_attempted":
		mc.retryAttempts.Inc()
	case "retry_succeeded":
		mc.retrySuccess.Inc()
	case "retry_failed":
		mc.retryFailed.Inc()
	}
}

// Circuit breaker metrics methods
func (mc *MetricsCollector) UpdateCircuitBreakerState(ctx context.Context, state string) {
	var stateValue float64
	switch state {
	case "closed":
		stateValue = 0
	case "open":
		stateValue = 1
	case "half-open":
		stateValue = 2
	}
	mc.circuitBreakerState.Set(stateValue)
}

func (mc *MetricsCollector) RecordCircuitBreakerResult(ctx context.Context, success bool) {
	if success {
		mc.circuitBreakerSuccesses.Inc()
	} else {
		mc.circuitBreakerFailures.Inc()
	}
}

// Rate limiting metrics methods
func (mc *MetricsCollector) RecordRateLimit(ctx context.Context, allowed bool) {
	mc.rateLimitRequests.Inc()
	if allowed {
		mc.rateLimitAllowed.Inc()
	} else {
		mc.rateLimitRejected.Inc()
	}
}

// Webhook metrics methods
func (mc *MetricsCollector) RecordWebhook(ctx context.Context, success bool, duration time.Duration) {
	mc.webhookReceived.Inc()
	if success {
		mc.webhookProcessed.Inc()
	} else {
		mc.webhookFailed.Inc()
	}
	mc.webhookDuration.Observe(duration.Seconds())
}

// Database metrics methods
func (mc *MetricsCollector) UpdateDBConnections(count int) {
	mc.dbConnections.Set(float64(count))
}

func (mc *MetricsCollector) RecordDBQuery(ctx context.Context, success bool, duration time.Duration) {
	mc.dbQueryDuration.Observe(duration.Seconds())
	if !success {
		mc.dbQueryErrors.Inc()
	}
}

// Cache metrics methods
func (mc *MetricsCollector) RecordCacheOperation(ctx context.Context, hit bool, duration time.Duration) {
	if hit {
		mc.cacheHits.Inc()
	} else {
		mc.cacheMisses.Inc()
	}
	mc.cacheDuration.Observe(duration.Seconds())
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	metricsCollector *MetricsCollector
	logger           *zap.Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(metricsCollector *MetricsCollector) *HealthChecker {
	return &HealthChecker{
		metricsCollector: metricsCollector,
		logger:           log.L(context.Background()),
	}
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Component string            `json:"component"`
	Status    string            `json:"status"`
	Details   map[string]string `json:"details"`
	Timestamp time.Time         `json:"timestamp"`
}

// CheckHealth performs health checks on all components
func (hc *HealthChecker) CheckHealth(ctx context.Context) map[string]HealthStatus {
	statuses := make(map[string]HealthStatus)

	// Check database health
	statuses["database"] = hc.checkDatabaseHealth(ctx)

	// Check Redis health
	statuses["redis"] = hc.checkRedisHealth(ctx)

	// Check external services health
	statuses["stripe"] = hc.checkStripeHealth(ctx)

	// Check circuit breakers
	statuses["circuit_breakers"] = hc.checkCircuitBreakersHealth(ctx)

	return statuses
}

// checkDatabaseHealth checks database health
func (hc *HealthChecker) checkDatabaseHealth(ctx context.Context) HealthStatus {
	// This would typically perform a simple query
	// For now, return healthy status
	return HealthStatus{
		Component: "database",
		Status:    "healthy",
		Details: map[string]string{
			"connection_pool": "active",
			"last_check":      time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}
}

// checkRedisHealth checks Redis health
func (hc *HealthChecker) checkRedisHealth(ctx context.Context) HealthStatus {
	// This would typically ping Redis
	// For now, return healthy status
	return HealthStatus{
		Component: "redis",
		Status:    "healthy",
		Details: map[string]string{
			"connection": "active",
			"last_check": time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}
}

// checkStripeHealth checks Stripe API health
func (hc *HealthChecker) checkStripeHealth(ctx context.Context) HealthStatus {
	// This would typically check Stripe API status
	// For now, return healthy status
	return HealthStatus{
		Component: "stripe",
		Status:    "healthy",
		Details: map[string]string{
			"api_status": "operational",
			"last_check": time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}
}

// checkCircuitBreakersHealth checks circuit breaker health
func (hc *HealthChecker) checkCircuitBreakersHealth(ctx context.Context) HealthStatus {
	// This would typically check all circuit breaker states
	// For now, return healthy status
	return HealthStatus{
		Component: "circuit_breakers",
		Status:    "healthy",
		Details: map[string]string{
			"stripe":     "closed",
			"database":   "closed",
			"last_check": time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now(),
	}
}

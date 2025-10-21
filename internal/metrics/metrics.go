package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// gRPC metrics
	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Payment metrics
	PaymentCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_created_total",
			Help: "Total number of payments created",
		},
		[]string{"status", "currency"},
	)

	PaymentAmount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_amount",
			Help:    "Payment amount distribution",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"currency"},
	)

	// Entitlement metrics
	EntitlementChecked = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "entitlement_checked_total",
			Help: "Total number of entitlement checks",
		},
		[]string{"feature_code", "allowed"},
	)

	EntitlementCacheHit = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "entitlement_cache_hit_total",
			Help: "Total number of entitlement cache hits",
		},
	)

	EntitlementCacheMiss = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "entitlement_cache_miss_total",
			Help: "Total number of entitlement cache misses",
		},
	)

	// Webhook metrics
	WebhookReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_received_total",
			Help: "Total number of webhooks received",
		},
		[]string{"event_type", "status"},
	)

	WebhookProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_processed_total",
			Help: "Total number of webhooks processed",
		},
		[]string{"event_type"},
	)

	// Database metrics
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	DatabaseConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	// Redis metrics
	RedisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	RedisOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_operation_duration_seconds",
			Help:    "Redis operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// Stripe metrics
	StripeAPICalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stripe_api_calls_total",
			Help: "Total number of Stripe API calls",
		},
		[]string{"operation", "status"},
	)

	StripeAPIDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stripe_api_duration_seconds",
			Help:    "Stripe API call duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "component"},
	)
)

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request
func RecordGRPCRequest(method, status string, duration time.Duration) {
	GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordPaymentCreated records a payment creation
func RecordPaymentCreated(status, currency string, amount float64) {
	PaymentCreated.WithLabelValues(status, currency).Inc()
	PaymentAmount.WithLabelValues(currency).Observe(amount)
}

// RecordEntitlementCheck records an entitlement check
func RecordEntitlementCheck(featureCode string, allowed bool) {
	allowedStr := "false"
	if allowed {
		allowedStr = "true"
	}
	EntitlementChecked.WithLabelValues(featureCode, allowedStr).Inc()
}

// RecordEntitlementCacheHit records an entitlement cache hit
func RecordEntitlementCacheHit() {
	EntitlementCacheHit.Inc()
}

// RecordEntitlementCacheMiss records an entitlement cache miss
func RecordEntitlementCacheMiss() {
	EntitlementCacheMiss.Inc()
}

// RecordWebhookReceived records a webhook reception
func RecordWebhookReceived(eventType, status string) {
	WebhookReceived.WithLabelValues(eventType, status).Inc()
}

// RecordWebhookProcessed records a webhook processing
func RecordWebhookProcessed(eventType string) {
	WebhookProcessed.WithLabelValues(eventType).Inc()
}

// RecordDatabaseQuery records a database query
func RecordDatabaseQuery(operation string, duration time.Duration) {
	DatabaseQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordRedisOperation records a Redis operation
func RecordRedisOperation(operation, status string, duration time.Duration) {
	RedisOperationsTotal.WithLabelValues(operation, status).Inc()
	RedisOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordStripeAPICall records a Stripe API call
func RecordStripeAPICall(operation, status string, duration time.Duration) {
	StripeAPICalls.WithLabelValues(operation, status).Inc()
	StripeAPIDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordError records an error
func RecordError(errorType, component string) {
	ErrorsTotal.WithLabelValues(errorType, component).Inc()
}

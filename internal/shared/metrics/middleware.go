package metrics

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/jia-app/paymentservice/internal/shared/log"
)

// MetricsInterceptor provides gRPC interceptors for metrics collection
type MetricsInterceptor struct {
	metricsCollector *MetricsCollector
	logger           *zap.Logger
}

// NewMetricsInterceptor creates a new metrics interceptor
func NewMetricsInterceptor(metricsCollector *MetricsCollector) *MetricsInterceptor {
	return &MetricsInterceptor{
		metricsCollector: metricsCollector,
		logger:           log.L(context.Background()),
	}
}

// UnaryServerInterceptor returns a unary server interceptor that collects metrics
func (mi *MetricsInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Record request start
		mi.logger.Debug("Processing gRPC request",
			zap.String("method", info.FullMethod),
			zap.Time("start_time", start))

		// Call the actual handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics based on method type
		mi.recordMethodMetrics(ctx, info.FullMethod, err, duration)

		// Log completion
		mi.logger.Debug("Completed gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err))

		return resp, err
	}
}

// StreamServerInterceptor returns a stream server interceptor that collects metrics
func (mi *MetricsInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Record request start
		mi.logger.Debug("Processing gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Time("start_time", start))

		// Call the actual handler
		err := handler(srv, ss)

		// Calculate duration
		duration := time.Since(start)

		// Record metrics based on method type
		mi.recordMethodMetrics(context.Background(), info.FullMethod, err, duration)

		// Log completion
		mi.logger.Debug("Completed gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.Error(err))

		return err
	}
}

// recordMethodMetrics records metrics based on the gRPC method
func (mi *MetricsInterceptor) recordMethodMetrics(ctx context.Context, method string, err error, duration time.Duration) {
	success := err == nil

	// Record general request metrics
	if success {
		mi.metricsCollector.RecordRateLimit(ctx, true)
	} else {
		mi.metricsCollector.RecordRateLimit(ctx, false)
	}

	// Record specific metrics based on method
	switch method {
	case "/payment.v1.PaymentService/ProcessPayment":
		mi.metricsCollector.RecordPayment(ctx, success, 0, duration)
	case "/payment.v1.PaymentService/CheckEntitlement":
		mi.metricsCollector.RecordEntitlementCheck(ctx, false, duration) // Assume cache miss for now
	case "/payment.v1.PaymentService/BulkCheckEntitlements":
		mi.metricsCollector.RecordEntitlementCheck(ctx, false, duration) // Assume cache miss for now
	case "/payment.v1.PaymentService/PaymentSuccessWebhook":
		mi.metricsCollector.RecordWebhook(ctx, success, duration)
	default:
		// For other methods, just record general metrics
		mi.logger.Debug("Unknown method for metrics",
			zap.String("method", method))
	}
}

// DatabaseMetricsInterceptor provides database operation metrics
type DatabaseMetricsInterceptor struct {
	metricsCollector *MetricsCollector
	logger           *zap.Logger
}

// NewDatabaseMetricsInterceptor creates a new database metrics interceptor
func NewDatabaseMetricsInterceptor(metricsCollector *MetricsCollector) *DatabaseMetricsInterceptor {
	return &DatabaseMetricsInterceptor{
		metricsCollector: metricsCollector,
		logger:           log.L(context.Background()),
	}
}

// WrapQuery wraps a database query with metrics collection
func (dmi *DatabaseMetricsInterceptor) WrapQuery(ctx context.Context, queryName string, queryFunc func() error) error {
	start := time.Now()

	err := queryFunc()
	duration := time.Since(start)

	success := err == nil
	dmi.metricsCollector.RecordDBQuery(ctx, success, duration)

	if !success {
		dmi.logger.Warn("Database query failed",
			zap.String("query", queryName),
			zap.Duration("duration", duration),
			zap.Error(err))
	}

	return err
}

// CacheMetricsInterceptor provides cache operation metrics
type CacheMetricsInterceptor struct {
	metricsCollector *MetricsCollector
	logger           *zap.Logger
}

// NewCacheMetricsInterceptor creates a new cache metrics interceptor
func NewCacheMetricsInterceptor(metricsCollector *MetricsCollector) *CacheMetricsInterceptor {
	return &CacheMetricsInterceptor{
		metricsCollector: metricsCollector,
		logger:           log.L(context.Background()),
	}
}

// WrapCacheOperation wraps a cache operation with metrics collection
func (cmi *CacheMetricsInterceptor) WrapCacheOperation(ctx context.Context, operation string, hit bool, operationFunc func() error) error {
	start := time.Now()

	err := operationFunc()
	duration := time.Since(start)

	cmi.metricsCollector.RecordCacheOperation(ctx, hit, duration)

	if err != nil {
		cmi.logger.Warn("Cache operation failed",
			zap.String("operation", operation),
			zap.Bool("hit", hit),
			zap.Duration("duration", duration),
			zap.Error(err))
	}

	return err
}

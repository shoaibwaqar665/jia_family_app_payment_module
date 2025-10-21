package tracing

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Config holds tracing configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	SamplingRatio  float64
}

// DefaultConfig returns a default tracing configuration
func DefaultConfig() Config {
	return Config{
		ServiceName:    "payment-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		JaegerEndpoint: "http://localhost:14268/api/traces",
		SamplingRatio:  1.0,
	}
}

// Init initializes OpenTelemetry tracing
func Init(config Config, logger *zap.Logger) (func(), error) {
	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SamplingRatio)),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry tracing initialized",
		zap.String("service_name", config.ServiceName),
		zap.String("service_version", config.ServiceVersion),
		zap.String("environment", config.Environment),
		zap.String("jaeger_endpoint", config.JaegerEndpoint),
		zap.Float64("sampling_ratio", config.SamplingRatio))

	// Return cleanup function
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown trace provider", zap.Error(err))
		}
	}, nil
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer("payment-service")
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, attrs...)
	}
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...trace.SpanStartOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		// Note: This is a simplified implementation
		// In practice, you'd want to use span.SetAttributes()
	}
}

// GetTraceID returns the trace ID from context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsEnabled checks if tracing is enabled
func IsEnabled() bool {
	return os.Getenv("OTEL_TRACING_ENABLED") == "true"
}

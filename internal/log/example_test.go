package log

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

// Example usage of the context-aware logger
func ExampleL() {
	// Initialize logger for testing
	_ = Init("info")

	// Create base context
	ctx := context.Background()

	// Add request-scoped fields to context
	ctx = WithUserID(ctx, "user123")
	ctx = WithRequestID(ctx, "req456")
	ctx = WithTraceID(ctx, "trace789")

	// Log with context - will include user_id, request_id, and trace_id
	L(ctx).Info("Processing payment",
		zap.String("payment_id", "pay_123"),
		zap.Int64("amount", 1000))

	// Using convenience functions
	Info(ctx, "Payment processed successfully",
		zap.String("payment_id", "pay_123"),
		zap.String("status", "completed"))

	Error(ctx, "Payment failed",
		zap.String("payment_id", "pay_123"),
		zap.String("error", "insufficient funds"))
}

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	// Test WithUserID
	ctx = WithUserID(ctx, "test_user")
	if userID := ctx.Value(UserIDKey); userID != "test_user" {
		t.Errorf("Expected user_id to be 'test_user', got %v", userID)
	}

	// Test WithRequestID
	ctx = WithRequestID(ctx, "test_request")
	if requestID := ctx.Value(RequestIDKey); requestID != "test_request" {
		t.Errorf("Expected request_id to be 'test_request', got %v", requestID)
	}

	// Test WithTraceID
	ctx = WithTraceID(ctx, "test_trace")
	if traceID := ctx.Value(TraceIDKey); traceID != "test_trace" {
		t.Errorf("Expected trace_id to be 'test_trace', got %v", traceID)
	}
}

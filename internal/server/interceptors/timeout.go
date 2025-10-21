package interceptors

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/log"
)

// TimeoutInterceptor provides timeout middleware for gRPC
type TimeoutInterceptor struct {
	defaultTimeout time.Duration
	methodTimeouts map[string]time.Duration
}

// NewTimeoutInterceptor creates a new timeout interceptor
func NewTimeoutInterceptor(defaultTimeout time.Duration, methodTimeouts map[string]time.Duration) *TimeoutInterceptor {
	return &TimeoutInterceptor{
		defaultTimeout: defaultTimeout,
		methodTimeouts: methodTimeouts,
	}
}

// Unary returns a unary interceptor for timeout handling
func (i *TimeoutInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Determine timeout for this method
		timeout := i.getTimeoutForMethod(info.FullMethod)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Log timeout configuration
		log.Debug(ctx, "Request timeout configured",
			zap.String("method", info.FullMethod),
			zap.Duration("timeout", timeout))

		// Call the handler with timeout context
		resp, err := handler(ctx, req)

		// Check if the error is due to timeout
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Warn(ctx, "Request timeout exceeded",
					zap.String("method", info.FullMethod),
					zap.Duration("timeout", timeout))
				return nil, status.Errorf(codes.DeadlineExceeded, "request timeout exceeded")
			}
		}

		return resp, err
	}
}

// Stream returns a stream interceptor for timeout handling
func (i *TimeoutInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Determine timeout for this method
		timeout := i.getTimeoutForMethod(info.FullMethod)

		// Create context with timeout
		ctx, cancel := context.WithTimeout(stream.Context(), timeout)
		defer cancel()

		// Log timeout configuration
		log.Debug(ctx, "Stream timeout configured",
			zap.String("method", info.FullMethod),
			zap.Duration("timeout", timeout))

		// Wrap the stream with timeout context
		wrappedStream := &timeoutServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		// Call the handler with timeout context
		err := handler(srv, wrappedStream)

		// Check if the error is due to timeout
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				log.Warn(ctx, "Stream timeout exceeded",
					zap.String("method", info.FullMethod),
					zap.Duration("timeout", timeout))
				return status.Errorf(codes.DeadlineExceeded, "stream timeout exceeded")
			}
		}

		return err
	}
}

// getTimeoutForMethod returns the timeout duration for a specific method
func (i *TimeoutInterceptor) getTimeoutForMethod(method string) time.Duration {
	if timeout, exists := i.methodTimeouts[method]; exists {
		return timeout
	}
	return i.defaultTimeout
}

// timeoutServerStream wraps grpc.ServerStream to provide a timeout context
type timeoutServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context with timeout
func (w *timeoutServerStream) Context() context.Context {
	return w.ctx
}

// AddMethodTimeout adds a timeout for a specific method
func (i *TimeoutInterceptor) AddMethodTimeout(method string, timeout time.Duration) {
	if i.methodTimeouts == nil {
		i.methodTimeouts = make(map[string]time.Duration)
	}
	i.methodTimeouts[method] = timeout
}

// RemoveMethodTimeout removes a timeout for a specific method
func (i *TimeoutInterceptor) RemoveMethodTimeout(method string) {
	delete(i.methodTimeouts, method)
}

// GetMethodTimeout returns the timeout for a specific method
func (i *TimeoutInterceptor) GetMethodTimeout(method string) time.Duration {
	return i.getTimeoutForMethod(method)
}

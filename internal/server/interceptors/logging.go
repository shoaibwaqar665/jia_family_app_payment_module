package interceptors

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/log"
)

// LoggingInterceptor provides request logging middleware for gRPC
type LoggingInterceptor struct{}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{}
}

// Unary returns a unary interceptor for request logging
func (i *LoggingInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		ctx = log.WithRequestID(ctx, requestID)

		// Extract user_id from metadata if available
		if userID := extractUserIDFromMetadata(ctx); userID != "" {
			ctx = log.WithUserID(ctx, userID)
		}

		// Log request start
		log.Info(ctx, "gRPC request started",
			zap.String("method", info.FullMethod),
			zap.Time("start_time", start))

		// Call the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		if err != nil {
			st, _ := status.FromError(err)
			log.Error(ctx, "gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", st.Code().String()),
				zap.String("error", st.Message()))
		} else {
			log.Info(ctx, "gRPC request completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", codes.OK.String()))
		}

		return resp, err
	}
}

// Stream returns a stream interceptor for request logging
func (i *LoggingInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Generate request ID
		requestID := uuid.New().String()
		ctx := log.WithRequestID(stream.Context(), requestID)

		// Wrap the stream with our context
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		// Log stream start
		log.Info(ctx, "gRPC stream started",
			zap.String("method", info.FullMethod),
			zap.Time("start_time", start))

		// Call the handler
		err := handler(srv, wrappedStream)

		// Calculate duration
		duration := time.Since(start)

		// Log stream completion
		if err != nil {
			st, _ := status.FromError(err)
			log.Error(ctx, "gRPC stream failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", st.Code().String()),
				zap.String("error", st.Message()))
		} else {
			log.Info(ctx, "gRPC stream completed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.String("code", codes.OK.String()))
		}

		return err
	}
}

// wrappedServerStream wraps grpc.ServerStream to provide a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// extractUserIDFromMetadata extracts user ID from gRPC metadata
func extractUserIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	// Check for user_id in metadata
	if userIDs := md.Get("user_id"); len(userIDs) > 0 {
		return userIDs[0]
	}

	// Check for authorization header and extract from JWT token
	if authHeaders := md.Get("authorization"); len(authHeaders) > 0 {
		authHeader := authHeaders[0]
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// In a real implementation, you would decode the JWT token here
			// For now, we'll just return a placeholder
			if token != "" {
				return "jwt_user_" + token[:min(len(token), 8)] // Use first 8 chars as placeholder
			}
		}
	}

	return ""
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

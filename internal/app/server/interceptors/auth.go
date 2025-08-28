package interceptors

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/shared/auth"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// AuthInterceptor provides authentication middleware for gRPC
type AuthInterceptor struct {
	// Whitelisted methods that don't require authentication
	whitelistedMethods map[string]bool
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{
		whitelistedMethods: map[string]bool{
			"/payment.v1.PaymentService/PaymentSuccessWebhook": true,
		},
	}
}

// Unary returns a unary interceptor for authentication
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if method is whitelisted
		if i.whitelistedMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Authenticate the request
		userID, err := i.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		// Inject user_id into context and add to logs
		ctx = log.WithUserID(ctx, userID)

		// Log successful authentication
		log.Info(ctx, "Request authenticated",
			zap.String("method", info.FullMethod),
			zap.String("user_id", userID))

		return handler(ctx, req)
	}
}

// Stream returns a stream interceptor for authentication
func (i *AuthInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check if method is whitelisted
		if i.whitelistedMethods[info.FullMethod] {
			return handler(srv, stream)
		}

		// Authenticate the stream
		userID, err := i.authenticate(stream.Context())
		if err != nil {
			return err
		}

		// Create new context with user_id
		ctx := log.WithUserID(stream.Context(), userID)

		// Log successful authentication
		log.Info(ctx, "Stream authenticated",
			zap.String("method", info.FullMethod),
			zap.String("user_id", userID))

		// Wrap the stream with authenticated context
		wrappedStream := &authenticatedServerStream{
			ServerStream: stream,
			ctx:          ctx,
		}

		return handler(srv, wrappedStream)
	}
}

// authenticate performs authentication check and returns user ID
func (i *AuthInterceptor) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	// Look for authorization metadata with key "better-auth-token"
	authTokens := md.Get("better-auth-token")
	if len(authTokens) == 0 {
		// Fallback to standard "authorization" header
		authTokens = md.Get("authorization")
	}

	if len(authTokens) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	token := authTokens[0]
	if token == "" {
		return "", status.Errorf(codes.Unauthenticated, "invalid authorization token")
	}

	// Validate token using the auth validator
	userID, err := auth.Validate(ctx, token)
	if err != nil {
		return "", status.Errorf(codes.Unauthenticated, "token validation failed: %v", err)
	}

	return userID, nil
}

// authenticatedServerStream wraps grpc.ServerStream to provide authenticated context
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the wrapped context with user_id
func (w *authenticatedServerStream) Context() context.Context {
	return w.ctx
}

// AddWhitelistedMethod adds a method to the whitelist
func (i *AuthInterceptor) AddWhitelistedMethod(method string) {
	i.whitelistedMethods[method] = true
}

// RemoveWhitelistedMethod removes a method from the whitelist
func (i *AuthInterceptor) RemoveWhitelistedMethod(method string) {
	delete(i.whitelistedMethods, method)
}

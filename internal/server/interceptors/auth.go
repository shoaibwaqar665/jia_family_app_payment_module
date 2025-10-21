package interceptors

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/auth"
	"github.com/jia-app/paymentservice/internal/log"
)

// AuthInterceptor provides authentication middleware for gRPC
type AuthInterceptor struct {
	validator          auth.Validator
	whitelistedMethods map[string]bool
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor(validator auth.Validator, whitelistedMethods []string) *AuthInterceptor {
	whitelistMap := make(map[string]bool)
	for _, method := range whitelistedMethods {
		whitelistMap[method] = true
	}

	return &AuthInterceptor{
		validator:          validator,
		whitelistedMethods: whitelistMap,
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

	// Look for authorization metadata with standard "authorization" header
	authTokens := md.Get("authorization")
	if len(authTokens) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}

	// Security: Only use the first authorization header to prevent header injection
	token := authTokens[0]
	if token == "" {
		return "", status.Errorf(codes.Unauthenticated, "invalid authorization token")
	}

	// Extract token from Bearer format
	token = auth.ExtractTokenFromAuthHeader(token)
	if token == "" {
		return "", status.Errorf(codes.Unauthenticated, "invalid authorization token format")
	}

	// Security: Validate token length to prevent extremely short tokens
	if len(token) < 10 {
		return "", status.Errorf(codes.Unauthenticated, "authorization token too short")
	}

	// Security: Set timeout for token validation to prevent hanging
	validationCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Validate token using the provided validator
	userID, err := i.validator.Validate(validationCtx, token)
	if err != nil {
		// Security: Don't expose internal validation errors
		return "", status.Errorf(codes.Unauthenticated, "token validation failed")
	}

	// Security: Validate user ID is not empty
	if strings.TrimSpace(userID) == "" {
		return "", status.Errorf(codes.Unauthenticated, "invalid user ID")
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

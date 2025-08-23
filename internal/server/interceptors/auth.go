package interceptors

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor provides authentication middleware for gRPC
type AuthInterceptor struct {
	// TODO: Add authentication configuration
}

// NewAuthInterceptor creates a new authentication interceptor
func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{}
}

// Unary returns a unary interceptor for authentication
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// TODO: Implement authentication logic
		if err := i.authenticate(ctx); err != nil {
			return nil, err
		}
		
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
		// TODO: Implement authentication logic
		if err := i.authenticate(stream.Context()); err != nil {
			return err
		}
		
		return handler(srv, stream)
	}
}

// authenticate performs authentication check
func (i *AuthInterceptor) authenticate(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}
	
	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return status.Errorf(codes.Unauthenticated, "authorization token is not provided")
	}
	
	// TODO: Validate token
	token := authHeader[0]
	if token == "" {
		return status.Errorf(codes.Unauthenticated, "invalid authorization token")
	}
	
	// TODO: Add proper token validation logic
	fmt.Printf("Authenticated with token: %s\n", token)
	
	return nil
}

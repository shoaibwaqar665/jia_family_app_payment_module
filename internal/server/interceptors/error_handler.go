package interceptors

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/log"
)

// ErrorHandlerInterceptor provides error handling middleware for gRPC
type ErrorHandlerInterceptor struct{}

// NewErrorHandlerInterceptor creates a new error handler interceptor
func NewErrorHandlerInterceptor() *ErrorHandlerInterceptor {
	return &ErrorHandlerInterceptor{}
}

// Unary returns a unary interceptor for error handling
func (i *ErrorHandlerInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		resp, err := handler(ctx, req)

		if err != nil {
			// Handle different types of errors
			handledErr := i.handleError(ctx, err, info.FullMethod)
			return resp, handledErr
		}

		return resp, nil
	}
}

// Stream returns a stream interceptor for error handling
func (i *ErrorHandlerInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		err := handler(srv, stream)

		if err != nil {
			// Handle different types of errors
			handledErr := i.handleError(stream.Context(), err, info.FullMethod)
			return handledErr
		}

		return nil
	}
}

// handleError processes and converts errors to appropriate gRPC status codes
func (i *ErrorHandlerInterceptor) handleError(ctx context.Context, err error, method string) error {
	if err == nil {
		return nil
	}

	// Log the error for debugging
	log.Error(ctx, "Error occurred in gRPC method",
		zap.String("method", method),
		zap.Error(err))

	// Check if it's already a gRPC status error
	if st, ok := status.FromError(err); ok {
		return st.Err()
	}

	// Handle domain-specific errors
	if domainErr := domain.GetDomainError(err); domainErr != nil {
		switch domainErr.Code {
		case domain.ErrCodeNotFound:
			return status.Errorf(codes.NotFound, domainErr.Message)
		case domain.ErrCodeInvalidInput:
			return status.Errorf(codes.InvalidArgument, domainErr.Message)
		case domain.ErrCodeUnauthorized:
			return status.Errorf(codes.Unauthenticated, domainErr.Message)
		case domain.ErrCodeAlreadyExists:
			return status.Errorf(codes.AlreadyExists, domainErr.Message)
		case domain.ErrCodeInvalidState:
			return status.Errorf(codes.FailedPrecondition, domainErr.Message)
		case domain.ErrCodeInsufficientFunds:
			return status.Errorf(codes.FailedPrecondition, domainErr.Message)
		case domain.ErrCodePaymentFailed:
			return status.Errorf(codes.FailedPrecondition, domainErr.Message)
		case domain.ErrCodeInternal:
			return status.Errorf(codes.Internal, domainErr.Message)
		default:
			return status.Errorf(codes.Internal, domainErr.Message)
		}
	}

	// Handle context errors
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return status.Errorf(codes.DeadlineExceeded, "request timeout")
	case errors.Is(err, context.Canceled):
		return status.Errorf(codes.Canceled, "request canceled")
	}

	// Handle HTTP status code errors (if any)
	if httpErr, ok := err.(*HTTPError); ok {
		return i.convertHTTPError(httpErr)
	}

	// Default to internal server error
	return status.Errorf(codes.Internal, "internal server error")
}

// convertHTTPError converts HTTP status codes to gRPC status codes
func (i *ErrorHandlerInterceptor) convertHTTPError(err *HTTPError) error {
	switch err.StatusCode {
	case http.StatusBadRequest:
		return status.Errorf(codes.InvalidArgument, err.Message)
	case http.StatusUnauthorized:
		return status.Errorf(codes.Unauthenticated, err.Message)
	case http.StatusForbidden:
		return status.Errorf(codes.PermissionDenied, err.Message)
	case http.StatusNotFound:
		return status.Errorf(codes.NotFound, err.Message)
	case http.StatusConflict:
		return status.Errorf(codes.AlreadyExists, err.Message)
	case http.StatusTooManyRequests:
		return status.Errorf(codes.ResourceExhausted, err.Message)
	case http.StatusInternalServerError:
		return status.Errorf(codes.Internal, err.Message)
	case http.StatusBadGateway:
		return status.Errorf(codes.Unavailable, err.Message)
	case http.StatusServiceUnavailable:
		return status.Errorf(codes.Unavailable, err.Message)
	case http.StatusGatewayTimeout:
		return status.Errorf(codes.DeadlineExceeded, err.Message)
	default:
		return status.Errorf(codes.Internal, err.Message)
	}
}

// HTTPError represents an HTTP error that can be converted to gRPC status
type HTTPError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	return e.Message
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
}

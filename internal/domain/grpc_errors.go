package domain

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorType represents the type of error for proper gRPC status mapping
type ErrorType int

const (
	ErrorTypeInvalidArgument ErrorType = iota
	ErrorTypeNotFound
	ErrorTypeAlreadyExists
	ErrorTypePermissionDenied
	ErrorTypeUnauthenticated
	ErrorTypeInternal
	ErrorTypeUnavailable
)

// GRPCError wraps domain errors with proper gRPC status codes
type GRPCError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *GRPCError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *GRPCError) Unwrap() error {
	return e.Cause
}

// ToGRPCStatus converts the error to appropriate gRPC status
func (e *GRPCError) ToGRPCStatus() *status.Status {
	switch e.Type {
	case ErrorTypeInvalidArgument:
		return status.New(codes.InvalidArgument, e.Message)
	case ErrorTypeNotFound:
		return status.New(codes.NotFound, e.Message)
	case ErrorTypeAlreadyExists:
		return status.New(codes.AlreadyExists, e.Message)
	case ErrorTypePermissionDenied:
		return status.New(codes.PermissionDenied, e.Message)
	case ErrorTypeUnauthenticated:
		return status.New(codes.Unauthenticated, e.Message)
	case ErrorTypeUnavailable:
		return status.New(codes.Unavailable, e.Message)
	case ErrorTypeInternal:
		fallthrough
	default:
		return status.New(codes.Internal, e.Message)
	}
}

// NewGRPCError creates a new GRPCError
func NewGRPCError(errorType ErrorType, message string, cause error) *GRPCError {
	return &GRPCError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// SanitizeError converts any error to a safe gRPC error without exposing internal details
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a GRPCError, return it
	var grpcErr *GRPCError
	if errors.As(err, &grpcErr) {
		return grpcErr
	}

	// Check for common domain errors and map them appropriately
	switch {
	case errors.Is(err, ErrPaymentNotFound):
		return NewGRPCError(ErrorTypeNotFound, "Payment not found", err)
	case errors.Is(err, ErrInvalidPaymentID):
		return NewGRPCError(ErrorTypeInvalidArgument, "Invalid payment ID", err)
	case errors.Is(err, ErrInvalidAmount):
		return NewGRPCError(ErrorTypeInvalidArgument, "Invalid payment amount", err)
	case errors.Is(err, ErrInvalidCurrency):
		return NewGRPCError(ErrorTypeInvalidArgument, "Invalid currency", err)
	case errors.Is(err, ErrPaymentAlreadyExists):
		return NewGRPCError(ErrorTypeAlreadyExists, "Payment already exists", err)
	case errors.Is(err, ErrUnauthorized):
		return NewGRPCError(ErrorTypeUnauthenticated, "Unauthorized", err)
	case errors.Is(err, ErrPermissionDenied):
		return NewGRPCError(ErrorTypePermissionDenied, "Permission denied", err)
	default:
		// For any other error, return a generic internal error without exposing details
		return NewGRPCError(ErrorTypeInternal, "Internal server error", nil)
	}
}

// Helper functions for common error types
func NewGRPCInvalidArgumentError(message string, cause error) *GRPCError {
	return NewGRPCError(ErrorTypeInvalidArgument, message, cause)
}

func NewGRPCNotFoundError(message string, cause error) *GRPCError {
	return NewGRPCError(ErrorTypeNotFound, message, cause)
}

func NewGRPCInternalError(message string) *GRPCError {
	return NewGRPCError(ErrorTypeInternal, message, nil)
}

func NewGRPCUnauthenticatedError(message string, cause error) *GRPCError {
	return NewGRPCError(ErrorTypeUnauthenticated, message, cause)
}

func NewGRPCPermissionDeniedError(message string, cause error) *GRPCError {
	return NewGRPCError(ErrorTypePermissionDenied, message, cause)
}

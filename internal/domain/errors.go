package domain

import (
	"fmt"
)

// DomainError represents a domain-specific error
type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface
func (e DomainError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common domain error codes
const (
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeAlreadyExists     = "ALREADY_EXISTS"
	ErrCodeInvalidState      = "INVALID_STATE"
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrCodePaymentFailed     = "PAYMENT_FAILED"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeInternal          = "INTERNAL_ERROR"
)

// NewInvalidInputError creates a new invalid input error
func NewInvalidInputError(message, details string) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidInput,
		Message: message,
		Details: details,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, id string) *DomainError {
	return &DomainError{
		Code:    ErrCodeNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Details: fmt.Sprintf("ID: %s", id),
	}
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(resource, id string) *DomainError {
	return &DomainError{
		Code:    ErrCodeAlreadyExists,
		Message: fmt.Sprintf("%s already exists", resource),
		Details: fmt.Sprintf("ID: %s", id),
	}
}

// NewInvalidStateError creates a new invalid state error
func NewInvalidStateError(message, details string) *DomainError {
	return &DomainError{
		Code:    ErrCodeInvalidState,
		Message: message,
		Details: details,
	}
}

// NewInsufficientFundsError creates a new insufficient funds error
func NewInsufficientFundsError(amount, available int64) *DomainError {
	return &DomainError{
		Code:    ErrCodeInsufficientFunds,
		Message: "Insufficient funds for payment",
		Details: fmt.Sprintf("Required: %d, Available: %d", amount, available),
	}
}

// NewPaymentFailedError creates a new payment failed error
func NewPaymentFailedError(reason string) *DomainError {
	return &DomainError{
		Code:    ErrCodePaymentFailed,
		Message: "Payment processing failed",
		Details: reason,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *DomainError {
	return &DomainError{
		Code:    ErrCodeUnauthorized,
		Message: message,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string) *DomainError {
	return &DomainError{
		Code:    ErrCodeInternal,
		Message: message,
	}
}

// IsDomainError checks if an error is a domain error
func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}

// GetDomainError extracts domain error from an error
func GetDomainError(err error) *DomainError {
	if domainErr, ok := err.(*DomainError); ok {
		return domainErr
	}
	return nil
}

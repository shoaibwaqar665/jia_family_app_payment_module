package repository

import (
	"context"

	"github.com/jia-app/paymentservice/internal/domain"
)

// PaymentRepository defines the interface for payment data operations
type PaymentRepository interface {
	// Create creates a new payment
	Create(ctx context.Context, payment *domain.Payment) error
	
	// GetByID retrieves a payment by ID
	GetByID(ctx context.Context, id string) (*domain.Payment, error)
	
	// GetByOrderID retrieves a payment by order ID
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	
	// GetByCustomerID retrieves payments by customer ID
	GetByCustomerID(ctx context.Context, customerID string, limit, offset int) ([]*domain.Payment, error)
	
	// Update updates an existing payment
	Update(ctx context.Context, payment *domain.Payment) error
	
	// UpdateStatus updates only the status of a payment
	UpdateStatus(ctx context.Context, id string, status string) error
	
	// Delete deletes a payment (soft delete)
	Delete(ctx context.Context, id string) error
	
	// List retrieves a list of payments with pagination
	List(ctx context.Context, limit, offset int) ([]*domain.Payment, error)
	
	// Count returns the total number of payments
	Count(ctx context.Context) (int64, error)
}

// Repository represents the main repository interface
type Repository interface {
	Payment() PaymentRepository
	Close() error
}

// Transaction represents a database transaction
type Transaction interface {
	Payment() PaymentRepository
	Commit() error
	Rollback() error
}

// TransactionManager manages database transactions
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(Transaction) error) error
}

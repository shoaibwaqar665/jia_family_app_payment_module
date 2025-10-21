package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/domain"
)

// ErrNotFound is returned when a record is not found
var ErrNotFound = errors.New("record not found")

// PaymentRepository defines the interface for payment data operations
type PaymentRepository interface {
	// Create creates a new payment
	Create(ctx context.Context, payment *domain.Payment) error

	// GetByID retrieves a payment by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)

	// GetByOrderID retrieves a payment by order ID
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)

	// GetByCustomerID retrieves payments by customer ID with pagination
	GetByCustomerID(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Payment, int, error)

	// Update updates an existing payment
	Update(ctx context.Context, payment *domain.Payment) error

	// UpdateStatus updates only the status of a payment
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error

	// Delete deletes a payment (soft delete)
	Delete(ctx context.Context, id string) error

	// List retrieves a list of payments with pagination
	List(ctx context.Context, limit, offset int) ([]*domain.Payment, int, error)

	// Count returns the total number of payments
	Count(ctx context.Context) (int64, error)
}

// PlanRepository defines the interface for plan operations
type PlanRepository interface {
	GetByID(ctx context.Context, id string) (domain.Plan, error)
	ListActive(ctx context.Context) ([]domain.Plan, error)
	Create(ctx context.Context, plan *domain.Plan) error
}

// EntitlementRepository defines the interface for entitlement operations
type EntitlementRepository interface {
	Check(ctx context.Context, userID, featureCode string) (domain.Entitlement, bool, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Entitlement, error)
	Insert(ctx context.Context, e domain.Entitlement) (domain.Entitlement, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateExpiry(ctx context.Context, id string, expiresAt *time.Time) error
}

// Repository represents the main repository interface
type Repository interface {
	Payment() PaymentRepository
	Plan() PlanRepository
	Entitlement() EntitlementRepository
	Close() error
}

// Transaction represents a database transaction
type Transaction interface {
	Payment() PaymentRepository
	Plan() PlanRepository
	Entitlement() EntitlementRepository
	WebhookEvents() WebhookEventsRepository
	Outbox() OutboxRepository
	Commit() error
	Rollback() error
}

// TransactionManager manages database transactions
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(Transaction) error) error
}

// WebhookEventsRepository defines the interface for webhook event operations
type WebhookEventsRepository interface {
	Insert(ctx context.Context, eventID, eventType, signature string, payload []byte) error
	GetByEventID(ctx context.Context, eventID string) (*domain.WebhookEvent, error)
	MarkProcessed(ctx context.Context, eventID string) error
}

// OutboxRepository defines the interface for transactional outbox operations
type OutboxRepository interface {
	Insert(ctx context.Context, eventType string, payload []byte) error
	GetPending(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, errorMessage string) error
}

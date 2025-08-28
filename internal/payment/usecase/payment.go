package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jia-app/paymentservice/internal/payment/domain"
)

// PaymentUseCase provides business logic for payment operations
type PaymentUseCase struct {
	paymentRepo PaymentRepository
}

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

// NewPaymentUseCase creates a new payment use case
func NewPaymentUseCase(paymentRepo PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo: paymentRepo,
	}
}

// CreatePayment creates a new payment
func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req *domain.PaymentRequest) (*domain.PaymentResponse, error) {
	// Validate request
	if err := uc.validatePaymentRequest(req); err != nil {
		return nil, err
	}

	// Create payment
	payment := &domain.Payment{
		ID:            uuid.New(),
		Amount:        req.Amount,
		Currency:      req.Currency,
		Status:        string(domain.PaymentStatusPending),
		PaymentMethod: req.PaymentMethod,
		CustomerID:    req.CustomerID,
		OrderID:       req.OrderID,
		Description:   req.Description,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save to repository
	if err := uc.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// TODO: Publish payment created event
	// TODO: Process payment with payment processor

	return &domain.PaymentResponse{
		ID:            payment.ID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		CustomerID:    payment.CustomerID,
		OrderID:       payment.OrderID,
		Description:   payment.Description,
		CreatedAt:     payment.CreatedAt,
	}, nil
}

// GetPayment retrieves a payment by ID
func (uc *PaymentUseCase) GetPayment(ctx context.Context, id string) (*domain.PaymentResponse, error) {
	payment, err := uc.paymentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment == nil {
		return nil, domain.NewNotFoundError("payment", id)
	}

	return &domain.PaymentResponse{
		ID:            payment.ID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		CustomerID:    payment.CustomerID,
		OrderID:       payment.OrderID,
		Description:   payment.Description,
		CreatedAt:     payment.CreatedAt,
	}, nil
}

// UpdatePaymentStatus updates the status of a payment
func (uc *PaymentUseCase) UpdatePaymentStatus(ctx context.Context, id string, status string) error {
	// Validate status
	tempPayment := &domain.Payment{Status: status}
	if !tempPayment.IsValidStatus() {
		return domain.NewInvalidInputError("invalid payment status", fmt.Sprintf("status: %s", status))
	}

	// Update status
	if err := uc.paymentRepo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// TODO: Publish payment status updated event

	return nil
}

// GetPaymentsByCustomer retrieves payments for a customer
func (uc *PaymentUseCase) GetPaymentsByCustomer(ctx context.Context, customerID string, limit, offset int) ([]*domain.PaymentResponse, error) {
	payments, err := uc.paymentRepo.GetByCustomerID(ctx, customerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer payments: %w", err)
	}

	responses := make([]*domain.PaymentResponse, len(payments))
	for i, payment := range payments {
		responses[i] = &domain.PaymentResponse{
			ID:            payment.ID,
			Amount:        payment.Amount,
			Currency:      payment.Currency,
			Status:        payment.Status,
			PaymentMethod: payment.PaymentMethod,
			CustomerID:    payment.CustomerID,
			OrderID:       payment.OrderID,
			Description:   payment.Description,
			CreatedAt:     payment.CreatedAt,
		}
	}

	return responses, nil
}

// validatePaymentRequest validates a payment request
func (uc *PaymentUseCase) validatePaymentRequest(req *domain.PaymentRequest) error {
	if req.Amount <= 0 {
		return domain.NewInvalidInputError("invalid amount", "amount must be greater than 0")
	}

	if req.Currency == "" || len(req.Currency) != 3 {
		return domain.NewInvalidInputError("invalid currency", "currency must be 3 characters")
	}

	if req.PaymentMethod == "" {
		return domain.NewInvalidInputError("invalid payment method", "payment method is required")
	}

	if req.CustomerID == "" {
		return domain.NewInvalidInputError("invalid customer ID", "customer ID is required")
	}

	if req.OrderID == "" {
		return domain.NewInvalidInputError("invalid order ID", "order ID is required")
	}

	return nil
}

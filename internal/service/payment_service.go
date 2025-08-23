package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/repository"
)

// PaymentService provides payment business logic
type PaymentService struct {
	repo repository.PaymentRepository
	// TODO: Add cache, event publisher, etc.
}

// NewPaymentService creates a new payment service
func NewPaymentService(repo repository.PaymentRepository) *PaymentService {
	return &PaymentService{
		repo: repo,
	}
}

// CreatePayment creates a new payment
func (s *PaymentService) CreatePayment(ctx context.Context, req *domain.PaymentRequest) (*domain.PaymentResponse, error) {
	// Validate request
	if err := s.validatePaymentRequest(req); err != nil {
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
	if err := s.repo.Create(ctx, payment); err != nil {
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
func (s *PaymentService) GetPayment(ctx context.Context, id string) (*domain.PaymentResponse, error) {
	payment, err := s.repo.GetByID(ctx, id)
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
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, id string, status string) error {
	// Validate status
	tempPayment := &domain.Payment{Status: status}
	if !tempPayment.IsValidStatus() {
		return domain.NewInvalidInputError("invalid payment status", fmt.Sprintf("status: %s", status))
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// TODO: Publish payment status updated event

	return nil
}

// GetPaymentsByCustomer retrieves payments for a customer
func (s *PaymentService) GetPaymentsByCustomer(ctx context.Context, customerID string, limit, offset int) ([]*domain.PaymentResponse, error) {
	payments, err := s.repo.GetByCustomerID(ctx, customerID, limit, offset)
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
func (s *PaymentService) validatePaymentRequest(req *domain.PaymentRequest) error {
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

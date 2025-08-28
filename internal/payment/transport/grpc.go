package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/usecase"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/events"
)

// PaymentService provides payment business logic and implements PaymentServiceServer
type PaymentService struct {
	config               *config.Config
	paymentUseCase       *usecase.PaymentUseCase
	entitlementUseCase   *usecase.EntitlementUseCase
	checkoutUseCase      *usecase.CheckoutUseCase
	cache                *cache.Cache
	entitlementPublisher events.EntitlementPublisher
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	config *config.Config,
	paymentUseCase *usecase.PaymentUseCase,
	entitlementUseCase *usecase.EntitlementUseCase,
	checkoutUseCase *usecase.CheckoutUseCase,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
) *PaymentService {
	return &PaymentService{
		config:               config,
		paymentUseCase:       paymentUseCase,
		entitlementUseCase:   entitlementUseCase,
		checkoutUseCase:      checkoutUseCase,
		cache:                cache,
		entitlementPublisher: entitlementPublisher,
	}
}

// CreatePayment creates a new payment
func (s *PaymentService) CreatePayment(ctx context.Context, req *domain.PaymentRequest) (*domain.PaymentResponse, error) {
	return s.paymentUseCase.CreatePayment(ctx, req)
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, id string) (*domain.PaymentResponse, error) {
	return s.paymentUseCase.GetPayment(ctx, id)
}

// UpdatePaymentStatus updates the status of a payment
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, id string, status string) error {
	return s.paymentUseCase.UpdatePaymentStatus(ctx, id, status)
}

// GetPaymentsByCustomer retrieves payments for a customer
func (s *PaymentService) GetPaymentsByCustomer(ctx context.Context, customerID string, limit, offset int) ([]*domain.PaymentResponse, error) {
	return s.paymentUseCase.GetPaymentsByCustomer(ctx, customerID, limit, offset)
}

// CheckEntitlement checks if a user has access to a specific feature
func (s *PaymentService) CheckEntitlement(ctx context.Context, userID, featureCode string) (*usecase.CheckEntitlementResponse, error) {
	return s.entitlementUseCase.CheckEntitlement(ctx, userID, featureCode)
}

// ListUserEntitlements lists all entitlements for a user
func (s *PaymentService) ListUserEntitlements(ctx context.Context, userID string) ([]*domain.Entitlement, error) {
	return s.entitlementUseCase.ListUserEntitlements(ctx, userID)
}

// CreateCheckoutSession creates a checkout session for a plan
func (s *PaymentService) CreateCheckoutSession(ctx context.Context, planID, userID string) (*usecase.CheckoutSessionResponse, error) {
	return s.checkoutUseCase.CreateCheckoutSession(ctx, planID, userID)
}

// PaymentSuccessWebhook handles payment success webhooks
func (s *PaymentService) PaymentSuccessWebhook(ctx context.Context, payload []byte, signature string) error {
	// Validate signature (stub implementation)
	if err := s.validateWebhookSignature(payload, signature); err != nil {
		return status.Error(codes.Unauthenticated, "invalid webhook signature")
	}

	// Parse payload (stub implementation)
	webhookData, err := s.parseWebhookPayload(payload)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid webhook payload")
	}

	// Create entitlement using use case
	_, err = s.entitlementUseCase.CreateEntitlement(ctx, webhookData.UserID, webhookData.FeatureCode, webhookData.PlanID, webhookData.ExpiresAt)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create entitlement: %v", err)
	}

	return nil
}

// Helper types for webhook handling
type WebhookData struct {
	UserID      string     `json:"user_id"`
	FeatureCode string     `json:"feature_code"`
	PlanID      uuid.UUID  `json:"plan_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Helper functions
func (s *PaymentService) validateWebhookSignature(payload []byte, signature string) error {
	// TODO: Implement actual signature validation
	// For now, just check that signature is not empty
	if signature == "" {
		return fmt.Errorf("missing signature")
	}
	return nil
}

func (s *PaymentService) parseWebhookPayload(payload []byte) (*WebhookData, error) {
	// TODO: Implement actual payload parsing
	// For now, return placeholder data
	return &WebhookData{
		UserID:      "user123",
		FeatureCode: "premium_feature",
		PlanID:      uuid.New(),
		ExpiresAt:   nil, // Never expires
	}, nil
}

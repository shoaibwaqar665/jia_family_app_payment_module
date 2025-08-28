package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/jia-app/paymentservice/api/payment/v1"
	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/usecase"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/events"
)

// PaymentService provides payment business logic and implements PaymentServiceServer
type PaymentService struct {
	paymentv1.UnimplementedPaymentServiceServer
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
func (s *PaymentService) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	// Convert proto request to domain request
	domainReq := &domain.PaymentRequest{
		Amount:        req.Amount,
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		CustomerID:    req.CustomerId,
		OrderID:       req.OrderId,
		Description:   req.Description,
	}

	// Call use case
	domainResp, err := s.paymentUseCase.CreatePayment(ctx, domainReq)
	if err != nil {
		return nil, err
	}

	// Convert domain response to proto response
	return &paymentv1.CreatePaymentResponse{
		Payment: &paymentv1.Payment{
			Id:            domainResp.ID.String(),
			Amount:        domainResp.Amount,
			Currency:      domainResp.Currency,
			Status:        domainResp.Status,
			PaymentMethod: domainResp.PaymentMethod,
			CustomerId:    domainResp.CustomerID,
			OrderId:       domainResp.OrderID,
			Description:   domainResp.Description,
			CreatedAt:     timestamppb.New(domainResp.CreatedAt),
		},
	}, nil
}

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	// Call use case
	domainResp, err := s.paymentUseCase.GetPayment(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// Convert domain response to proto response
	return &paymentv1.GetPaymentResponse{
		Payment: &paymentv1.Payment{
			Id:            domainResp.ID.String(),
			Amount:        domainResp.Amount,
			Currency:      domainResp.Currency,
			Status:        domainResp.Status,
			PaymentMethod: domainResp.PaymentMethod,
			CustomerId:    domainResp.CustomerID,
			OrderId:       domainResp.OrderID,
			Description:   domainResp.Description,
			CreatedAt:     timestamppb.New(domainResp.CreatedAt),
		},
	}, nil
}

// UpdatePaymentStatus updates the status of a payment
func (s *PaymentService) UpdatePaymentStatus(ctx context.Context, req *paymentv1.UpdatePaymentStatusRequest) (*paymentv1.UpdatePaymentStatusResponse, error) {
	// Call use case
	err := s.paymentUseCase.UpdatePaymentStatus(ctx, req.Id, req.Status)
	if err != nil {
		return nil, err
	}

	// Return success response
	return &paymentv1.UpdatePaymentStatusResponse{
		Success: true,
	}, nil
}

// GetPaymentsByCustomer retrieves payments for a customer
func (s *PaymentService) GetPaymentsByCustomer(ctx context.Context, req *paymentv1.GetPaymentsByCustomerRequest) (*paymentv1.GetPaymentsByCustomerResponse, error) {
	// Call use case
	domainResps, err := s.paymentUseCase.GetPaymentsByCustomer(ctx, req.CustomerId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}

	// Convert domain responses to proto responses
	payments := make([]*paymentv1.Payment, len(domainResps))
	for i, domainResp := range domainResps {
		payments[i] = &paymentv1.Payment{
			Id:            domainResp.ID.String(),
			Amount:        domainResp.Amount,
			Currency:      domainResp.Currency,
			Status:        domainResp.Status,
			PaymentMethod: domainResp.PaymentMethod,
			CustomerId:    domainResp.CustomerID,
			OrderId:       domainResp.OrderID,
			Description:   domainResp.Description,
			CreatedAt:     timestamppb.New(domainResp.CreatedAt),
		}
	}

	// Return response
	return &paymentv1.GetPaymentsByCustomerResponse{
		Payments: payments,
		Total:    int32(len(payments)), // TODO: Get actual total count
	}, nil
}

// ListPayments retrieves a list of payments with pagination
func (s *PaymentService) ListPayments(ctx context.Context, req *paymentv1.ListPaymentsRequest) (*paymentv1.ListPaymentsResponse, error) {
	// TODO: Implement ListPayments use case
	// For now, return empty response
	return &paymentv1.ListPaymentsResponse{
		Payments: []*paymentv1.Payment{},
		Total:    0,
	}, nil
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
func (s *PaymentService) CreateCheckoutSession(ctx context.Context, planID, userID string, familyID *string, countryCode string) (*usecase.CheckoutSessionResponse, error) {
	return s.checkoutUseCase.CreateCheckoutSession(ctx, planID, userID, familyID, countryCode)
}

// ApplyWebhook applies a webhook result from billing provider
func (s *PaymentService) ApplyWebhook(ctx context.Context, wr usecase.WebhookResult) error {
	return s.checkoutUseCase.ApplyWebhook(ctx, wr)
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

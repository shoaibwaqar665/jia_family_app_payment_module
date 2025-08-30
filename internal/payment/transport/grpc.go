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
	"github.com/jia-app/paymentservice/internal/billing"
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
	pricingZoneUseCase   *usecase.PricingZoneUseCase
	cache                *cache.Cache
	entitlementPublisher events.EntitlementPublisher
	billingProvider      billing.Provider
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	config *config.Config,
	paymentUseCase *usecase.PaymentUseCase,
	entitlementUseCase *usecase.EntitlementUseCase,
	checkoutUseCase *usecase.CheckoutUseCase,
	pricingZoneUseCase *usecase.PricingZoneUseCase,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
	billingProvider billing.Provider,
) *PaymentService {
	return &PaymentService{
		config:               config,
		paymentUseCase:       paymentUseCase,
		entitlementUseCase:   entitlementUseCase,
		checkoutUseCase:      checkoutUseCase,
		pricingZoneUseCase:   pricingZoneUseCase,
		cache:                cache,
		entitlementPublisher: entitlementPublisher,
		billingProvider:      billingProvider,
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
	// Call use case
	domainResps, err := s.paymentUseCase.ListPayments(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, err
	}

	// Get total count
	total, err := s.paymentUseCase.CountPayments(ctx)
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
	return &paymentv1.ListPaymentsResponse{
		Payments: payments,
		Total:    int32(total),
	}, nil
}

// CreateCheckoutSession creates a checkout session for payment
func (s *PaymentService) CreateCheckoutSession(ctx context.Context, req *paymentv1.CreateCheckoutSessionRequest) (*paymentv1.CreateCheckoutSessionResponse, error) {
	// Convert plan ID to UUID
	planID, err := uuid.Parse(req.PlanId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid plan_id")
	}

	// Prepare billing request
	billingReq := billing.CreateCheckoutSessionRequest{
		PlanID:      planID,
		UserID:      req.UserId,
		CountryCode: req.CountryCode,
		BasePrice:   req.BasePrice,
		Currency:    req.Currency,
		SuccessURL:  req.SuccessUrl,
		CancelURL:   req.CancelUrl,
	}

	// Add family ID if provided
	if req.FamilyId != "" {
		billingReq.FamilyID = &req.FamilyId
	}

	// Create checkout session using billing provider
	session, err := s.billingProvider.CreateCheckoutSession(ctx, billingReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create checkout session: %v", err)
	}

	// Create a payment record in the database
	paymentReq := &domain.PaymentRequest{
		Amount:        req.BasePrice,
		Currency:      req.Currency,
		PaymentMethod: "credit_card",
		CustomerID:    req.UserId,
		OrderID:       session.SessionID, // Use session ID as order ID
		Description:   fmt.Sprintf("Checkout session for plan %s", req.PlanId),
	}

	_, err = s.paymentUseCase.CreatePayment(ctx, paymentReq)
	if err != nil {
		// Log error but don't fail the checkout session creation
		// The payment record can be created later via webhook
		fmt.Printf("Warning: Failed to create payment record: %v\n", err)
	}

	// Convert to protobuf response
	return &paymentv1.CreateCheckoutSessionResponse{
		SessionId: session.SessionID,
		Url:       session.URL,
		ExpiresAt: timestamppb.New(session.ExpiresAt),
	}, nil
}

// ProcessWebhook processes webhook events from payment providers
func (s *PaymentService) ProcessWebhook(ctx context.Context, req *paymentv1.ProcessWebhookRequest) (*paymentv1.ProcessWebhookResponse, error) {
	// Validate webhook signature
	if err := s.billingProvider.ValidateWebhook(ctx, req.Payload, req.Signature); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "webhook validation failed: %v", err)
	}

	// Parse webhook payload
	webhookResult, err := s.billingProvider.ParseWebhook(ctx, req.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse webhook: %v", err)
	}

	// Apply webhook result (create entitlement)
	if err := s.checkoutUseCase.ApplyWebhook(ctx, *webhookResult); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to apply webhook: %v", err)
	}

	return &paymentv1.ProcessWebhookResponse{
		Success: true,
		Message: "Webhook processed successfully",
	}, nil
}

// ListUserEntitlements lists all entitlements for a user
func (s *PaymentService) ListUserEntitlements(ctx context.Context, userID string) ([]*domain.Entitlement, error) {
	return s.entitlementUseCase.ListUserEntitlements(ctx, userID)
}

// ApplyWebhook applies a webhook result from billing provider
func (s *PaymentService) ApplyWebhook(ctx context.Context, wr billing.WebhookResult) error {
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

// ListEntitlements retrieves entitlements for a user
func (s *PaymentService) ListEntitlements(ctx context.Context, req *paymentv1.ListEntitlementsRequest) (*paymentv1.ListEntitlementsResponse, error) {
	// Get entitlements from use case
	entitlements, err := s.entitlementUseCase.ListUserEntitlements(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list entitlements: %v", err)
	}

	// Convert to protobuf
	var pbEntitlements []*paymentv1.Entitlement
	for _, ent := range entitlements {
		pbEntitlement := &paymentv1.Entitlement{
			Id:          ent.ID.String(),
			UserId:      ent.UserID,
			FeatureCode: ent.FeatureCode,
			PlanId:      ent.PlanID.String(),
			Status:      ent.Status,
			GrantedAt:   timestamppb.New(ent.GrantedAt),
			CreatedAt:   timestamppb.New(ent.CreatedAt),
			UpdatedAt:   timestamppb.New(ent.UpdatedAt),
		}

		// Add optional fields
		if ent.FamilyID != nil && *ent.FamilyID != "" {
			pbEntitlement.FamilyId = *ent.FamilyID
		}
		if ent.SubscriptionID != nil && *ent.SubscriptionID != "" {
			pbEntitlement.SubscriptionId = *ent.SubscriptionID
		}
		if ent.ExpiresAt != nil {
			pbEntitlement.ExpiresAt = timestamppb.New(*ent.ExpiresAt)
		}

		pbEntitlements = append(pbEntitlements, pbEntitlement)
	}

	return &paymentv1.ListEntitlementsResponse{
		Entitlements: pbEntitlements,
		Total:        int32(len(pbEntitlements)),
	}, nil
}

// CheckEntitlement checks if a user has access to a feature
func (s *PaymentService) CheckEntitlement(ctx context.Context, req *paymentv1.CheckEntitlementRequest) (*paymentv1.CheckEntitlementResponse, error) {
	// Check entitlement using use case
	response, err := s.entitlementUseCase.CheckEntitlement(ctx, req.UserId, req.FeatureCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check entitlement: %v", err)
	}

	// Convert to protobuf
	pbResponse := &paymentv1.CheckEntitlementResponse{
		Allowed: response.Allowed,
	}

	if response.Entitlement != nil {
		pbEntitlement := &paymentv1.Entitlement{
			Id:          response.Entitlement.ID.String(),
			UserId:      response.Entitlement.UserID,
			FeatureCode: response.Entitlement.FeatureCode,
			PlanId:      response.Entitlement.PlanID.String(),
			Status:      response.Entitlement.Status,
			GrantedAt:   timestamppb.New(response.Entitlement.GrantedAt),
			CreatedAt:   timestamppb.New(response.Entitlement.CreatedAt),
			UpdatedAt:   timestamppb.New(response.Entitlement.UpdatedAt),
		}

		// Add optional fields
		if response.Entitlement.FamilyID != nil && *response.Entitlement.FamilyID != "" {
			pbEntitlement.FamilyId = *response.Entitlement.FamilyID
		}
		if response.Entitlement.SubscriptionID != nil && *response.Entitlement.SubscriptionID != "" {
			pbEntitlement.SubscriptionId = *response.Entitlement.SubscriptionID
		}
		if response.Entitlement.ExpiresAt != nil {
			pbEntitlement.ExpiresAt = timestamppb.New(*response.Entitlement.ExpiresAt)
		}

		pbResponse.Entitlement = pbEntitlement
	}

	return pbResponse, nil
}

// ListPricingZones retrieves all pricing zones
func (s *PaymentService) ListPricingZones(ctx context.Context, req *paymentv1.ListPricingZonesRequest) (*paymentv1.ListPricingZonesResponse, error) {
	// Get pricing zones from use case
	zones, err := s.pricingZoneUseCase.ListPricingZones(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list pricing zones: %v", err)
	}

	// Convert to protobuf
	var pbZones []*paymentv1.PricingZone
	for _, zone := range zones {
		pbZone := &paymentv1.PricingZone{
			Id:                zone.ID,
			IsoCode:           zone.ISOCode,
			Country:           zone.Country,
			Zone:              zone.Zone,
			ZoneName:          zone.ZoneName,
			PricingMultiplier: zone.PricingMultiplier,
			CreatedAt:         timestamppb.New(zone.CreatedAt),
			UpdatedAt:         timestamppb.New(zone.UpdatedAt),
		}
		pbZones = append(pbZones, pbZone)
	}

	return &paymentv1.ListPricingZonesResponse{
		PricingZones: pbZones,
	}, nil
}

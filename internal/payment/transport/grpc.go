package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/jia-app/paymentservice/api/payment/v1"
	"github.com/jia-app/paymentservice/internal/billing"
	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/usecase"
	"github.com/jia-app/paymentservice/internal/payment/webhook"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
	"github.com/jia-app/paymentservice/internal/shared/metrics"
)

// PaymentService provides payment business logic and implements PaymentServiceServer
type PaymentService struct {
	paymentv1.UnimplementedPaymentServiceServer
	config                 *config.Config
	paymentUseCase         *usecase.PaymentUseCase
	entitlementUseCase     *usecase.EntitlementUseCase
	bulkEntitlementUseCase *usecase.BulkEntitlementUseCase
	checkoutUseCase        *usecase.CheckoutUseCase
	pricingZoneUseCase     *usecase.PricingZoneUseCase
	cache                  *cache.Cache
	entitlementPublisher   events.EntitlementPublisher
	billingProvider        billing.Provider
	webhookValidator       *webhook.Validator
	webhookParser          *webhook.Parser
	metricsCollector       *metrics.MetricsCollector
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	config *config.Config,
	paymentUseCase *usecase.PaymentUseCase,
	entitlementUseCase *usecase.EntitlementUseCase,
	bulkEntitlementUseCase *usecase.BulkEntitlementUseCase,
	checkoutUseCase *usecase.CheckoutUseCase,
	pricingZoneUseCase *usecase.PricingZoneUseCase,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
	billingProvider billing.Provider,
	metricsCollector *metrics.MetricsCollector,
) *PaymentService {
	// Initialize webhook validator and parser
	webhookValidator := webhook.NewValidator(config.Billing.StripeWebhookSecret)
	webhookParser := webhook.NewParser()

	return &PaymentService{
		config:                 config,
		paymentUseCase:         paymentUseCase,
		entitlementUseCase:     entitlementUseCase,
		bulkEntitlementUseCase: bulkEntitlementUseCase,
		checkoutUseCase:        checkoutUseCase,
		pricingZoneUseCase:     pricingZoneUseCase,
		cache:                  cache,
		entitlementPublisher:   entitlementPublisher,
		billingProvider:        billingProvider,
		webhookValidator:       webhookValidator,
		webhookParser:          webhookParser,
		metricsCollector:       metricsCollector,
	}
}

// CreatePayment creates a new payment
func (s *PaymentService) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	start := time.Now()

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
	duration := time.Since(start)

	// Record metrics
	s.metricsCollector.RecordPayment(ctx, err == nil, float64(req.Amount)/100, duration)

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
	// Convert plan ID to UUID (handle both UUID and string plan IDs)
	var planID uuid.UUID
	if parsedUUID, err := uuid.Parse(req.PlanId); err == nil {
		// It's a valid UUID
		planID = parsedUUID
	} else {
		// It's a string plan ID, generate a deterministic UUID
		planID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(req.PlanId))
		log.Info(ctx, "Converted string plan ID to UUID",
			zap.String("plan_id_string", req.PlanId),
			zap.String("plan_id_uuid", planID.String()))
	}

	// Calculate adjusted price based on country code
	adjustedPrice := req.BasePrice
	if req.CountryCode != "" {
		// Get pricing zone for the country
		pricingZone, err := s.pricingZoneUseCase.GetPricingZoneByISOCode(ctx, req.CountryCode)
		if err == nil {
			adjustedPrice = pricingZone.CalculateAdjustedPrice(req.BasePrice)
			log.Info(ctx, "Applied dynamic pricing for checkout",
				zap.String("country_code", req.CountryCode),
				zap.String("zone", pricingZone.Zone),
				zap.String("zone_name", pricingZone.ZoneName),
				zap.Float64("multiplier", pricingZone.PricingMultiplier),
				zap.Float64("base_price", req.BasePrice),
				zap.Float64("adjusted_price", adjustedPrice))
		} else {
			log.Warn(ctx, "Pricing zone not found for checkout, using base price",
				zap.String("country_code", req.CountryCode),
				zap.Error(err))
		}
	}

	// Prepare billing request with adjusted price
	billingReq := billing.CreateCheckoutSessionRequest{
		PlanID:      planID,
		UserID:      req.UserId,
		CountryCode: req.CountryCode,
		BasePrice:   adjustedPrice, // Use adjusted price instead of base price
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

	// Create a payment record in the database with adjusted price
	paymentReq := &domain.PaymentRequest{
		Amount:        adjustedPrice, // Use adjusted price for payment record
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
	start := time.Now()

	// Validate webhook signature
	if err := s.webhookValidator.ValidateStripeWebhook(payload, signature); err != nil {
		log.Error(ctx, "Webhook signature validation failed", zap.Error(err))
		s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
		return status.Error(codes.Unauthenticated, "invalid webhook signature")
	}

	// Parse webhook payload
	webhookResult, err := s.webhookParser.ParseStripeWebhook(payload)
	if err != nil {
		log.Error(ctx, "Failed to parse webhook payload", zap.Error(err))
		s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
		return status.Error(codes.InvalidArgument, "invalid webhook payload")
	}

	// Convert to billing.WebhookResult
	billingResult := billing.WebhookResult{
		EventType:    "payment.succeeded", // Default event type
		SessionID:    "",                  // Will be set if available
		UserID:       webhookResult.UserID,
		FamilyID:     webhookResult.FamilyID,
		FeatureCode:  webhookResult.FeatureCode,
		PlanID:       webhookResult.PlanID,
		PlanIDString: webhookResult.PlanIDString,
		Amount:       float64(webhookResult.Amount),
		Currency:     webhookResult.Currency,
		Status:       webhookResult.Status,
		ExpiresAt:    webhookResult.ExpiresAt,
		Metadata:     webhookResult.Metadata,
	}

	// Apply webhook result using checkout use case
	if err := s.checkoutUseCase.ApplyWebhook(ctx, billingResult); err != nil {
		log.Error(ctx, "Failed to apply webhook result", zap.Error(err))
		s.metricsCollector.RecordWebhook(ctx, false, time.Since(start))
		return status.Errorf(codes.Internal, "failed to apply webhook: %v", err)
	}

	// Record successful webhook processing
	s.metricsCollector.RecordWebhook(ctx, true, time.Since(start))

	log.Info(ctx, "Webhook processed successfully",
		zap.String("user_id", webhookResult.UserID),
		zap.String("feature_code", webhookResult.FeatureCode),
		zap.String("status", webhookResult.Status))

	return nil
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
	start := time.Now()

	// Check entitlement using use case
	response, err := s.entitlementUseCase.CheckEntitlement(ctx, req.UserId, req.FeatureCode)
	duration := time.Since(start)

	// Record metrics (assuming cache miss for now - this would be improved with actual cache hit detection)
	s.metricsCollector.RecordEntitlementCheck(ctx, false, duration)

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

// BulkCheckEntitlements checks multiple entitlements in a single request
func (s *PaymentService) BulkCheckEntitlements(ctx context.Context, req *paymentv1.BulkCheckEntitlementsRequest) (*paymentv1.BulkCheckEntitlementsResponse, error) {
	// Convert proto request to use case request
	bulkReq := usecase.BulkCheckRequest{
		UserID: req.UserId,
		Checks: make([]usecase.BulkCheckItem, len(req.Checks)),
	}

	for i, check := range req.Checks {
		// Convert metadata from map[string]string to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range check.Metadata {
			metadata[k] = v
		}

		bulkReq.Checks[i] = usecase.BulkCheckItem{
			FeatureCode:  check.FeatureCode,
			Operation:    check.Operation,
			ResourceSize: check.ResourceSize,
			Metadata:     metadata,
		}
	}

	// Perform bulk check
	response, err := s.bulkEntitlementUseCase.BulkCheckEntitlements(ctx, bulkReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to perform bulk entitlement check: %v", err)
	}

	// Convert to protobuf response
	pbResults := make([]*paymentv1.BulkCheckResult, len(response.Results))
	for i, result := range response.Results {
		pbResult := &paymentv1.BulkCheckResult{
			FeatureCode: result.FeatureCode,
			Authorized:  result.Authorized,
			Reason:      result.Reason,
			UpgradeUrl:  result.UpgradeURL,
		}

		// Convert entitlement if present
		if result.Entitlement != nil {
			pbEntitlement := &paymentv1.Entitlement{
				Id:          result.Entitlement.ID.String(),
				UserId:      result.Entitlement.UserID,
				FeatureCode: result.Entitlement.FeatureCode,
				PlanId:      result.Entitlement.PlanID.String(),
				Status:      result.Entitlement.Status,
				GrantedAt:   timestamppb.New(result.Entitlement.GrantedAt),
				CreatedAt:   timestamppb.New(result.Entitlement.CreatedAt),
				UpdatedAt:   timestamppb.New(result.Entitlement.UpdatedAt),
			}

			if result.Entitlement.FamilyID != nil {
				pbEntitlement.FamilyId = *result.Entitlement.FamilyID
			}
			if result.Entitlement.SubscriptionID != nil {
				pbEntitlement.SubscriptionId = *result.Entitlement.SubscriptionID
			}
			if result.Entitlement.ExpiresAt != nil {
				pbEntitlement.ExpiresAt = timestamppb.New(*result.Entitlement.ExpiresAt)
			}

			pbResult.Entitlement = pbEntitlement
		}

		// Convert metadata
		if result.Metadata != nil {
			pbResult.Metadata = make(map[string]string)
			for k, v := range result.Metadata {
				if str, ok := v.(string); ok {
					pbResult.Metadata[k] = str
				}
			}
		}

		pbResults[i] = pbResult
	}

	// Convert summary
	pbSummary := &paymentv1.BulkCheckSummary{
		TotalChecks:      int32(response.Summary.TotalChecks),
		Authorized:       int32(response.Summary.Authorized),
		NotAuthorized:    int32(response.Summary.NotAuthorized),
		CacheHits:        int32(response.Summary.CacheHits),
		CacheMisses:      int32(response.Summary.CacheMisses),
		ProcessingTimeMs: response.Summary.ProcessingTime,
	}

	return &paymentv1.BulkCheckEntitlementsResponse{
		Results: pbResults,
		Summary: pbSummary,
	}, nil
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

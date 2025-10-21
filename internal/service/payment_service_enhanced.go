package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/billing"
	"github.com/jia-app/paymentservice/internal/cache"
	"github.com/jia-app/paymentservice/internal/circuitbreaker"
	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/events"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/repository"
	"github.com/jia-app/paymentservice/internal/retry"
)

// EnhancedPaymentService provides enhanced payment business logic with Stripe integration
type EnhancedPaymentService struct {
	config                *config.Config
	paymentRepo           repository.PaymentRepository
	planRepo              repository.PlanRepository
	entitlementRepo       repository.EntitlementRepository
	webhookEventsRepo     repository.WebhookEventsRepository
	outboxRepo            repository.OutboxRepository
	cache                 *cache.Cache
	entitlementPublisher  events.EntitlementPublisher
	paymentProvider       billing.PaymentProvider
	circuitBreakerManager *circuitbreaker.Manager
}

// NewEnhancedPaymentService creates a new enhanced payment service
func NewEnhancedPaymentService(
	config *config.Config,
	paymentRepo repository.PaymentRepository,
	planRepo repository.PlanRepository,
	entitlementRepo repository.EntitlementRepository,
	webhookEventsRepo repository.WebhookEventsRepository,
	outboxRepo repository.OutboxRepository,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
	paymentProvider billing.PaymentProvider,
	circuitBreakerManager *circuitbreaker.Manager,
) *EnhancedPaymentService {
	return &EnhancedPaymentService{
		config:                config,
		paymentRepo:           paymentRepo,
		planRepo:              planRepo,
		entitlementRepo:       entitlementRepo,
		webhookEventsRepo:     webhookEventsRepo,
		outboxRepo:            outboxRepo,
		cache:                 cache,
		entitlementPublisher:  entitlementPublisher,
		paymentProvider:       paymentProvider,
		circuitBreakerManager: circuitBreakerManager,
	}
}

// CreatePayment creates a new payment
func (s *EnhancedPaymentService) CreatePayment(ctx context.Context, req *domain.PaymentRequest) (*domain.Payment, error) {
	// Validate payment request
	if err := s.validatePaymentRequest(req); err != nil {
		return nil, err
	}

	// Create payment record
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

	// Save payment to database with circuit breaker protection and retry logic
	dbBreaker := s.circuitBreakerManager.GetOrCreate("database", circuitbreaker.DefaultConfig())
	retryConfig := retry.DefaultConfig()
	logger := log.L(ctx)

	err := dbBreaker.Execute(ctx, func() error {
		return retry.Do(ctx, retryConfig, logger, func() error {
			return s.paymentRepo.Create(ctx, payment)
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	return payment, nil
}

// GetPayment retrieves a payment by ID
func (s *EnhancedPaymentService) GetPayment(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	payment, err := s.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return payment, nil
}

// UpdatePaymentStatus updates the status of a payment
func (s *EnhancedPaymentService) UpdatePaymentStatus(ctx context.Context, id uuid.UUID, status string) error {
	if err := s.paymentRepo.UpdateStatus(ctx, id, status); err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("payment not found")
		}
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	return nil
}

// GetPaymentsByCustomer retrieves payments for a specific customer with pagination
func (s *EnhancedPaymentService) GetPaymentsByCustomer(ctx context.Context, customerID uuid.UUID, limit, offset int) ([]*domain.Payment, int, error) {
	payments, total, err := s.paymentRepo.GetByCustomerID(ctx, customerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get payments by customer: %w", err)
	}
	return payments, total, nil
}

// ListPayments retrieves a paginated list of payments
func (s *EnhancedPaymentService) ListPayments(ctx context.Context, limit, offset int) ([]*domain.Payment, int, error) {
	payments, total, err := s.paymentRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list payments: %w", err)
	}
	return payments, total, nil
}

// validatePaymentRequest validates a payment request
func (s *EnhancedPaymentService) validatePaymentRequest(req *domain.PaymentRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	if req.CustomerID == "" {
		return fmt.Errorf("customer ID is required")
	}
	if req.PaymentMethod == "" {
		return fmt.Errorf("payment method is required")
	}
	return nil
}

// CreateCheckoutSession creates a checkout session with Stripe
func (s *EnhancedPaymentService) CreateCheckoutSession(ctx context.Context, planID, userID string) (*CheckoutSessionResponse, error) {
	// Validate input
	if planID == "" {
		return nil, status.Error(codes.InvalidArgument, "plan_id is required")
	}
	if userID == "" {
		if contextUserID := extractUserIDFromContext(ctx); contextUserID != "" {
			userID = contextUserID
		} else {
			return nil, status.Error(codes.InvalidArgument, "user_id is required")
		}
	}

	// Get plan details
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "plan not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get plan: %v", err)
	}

	// Generate order ID
	orderID := fmt.Sprintf("order_%s_%d", userID, time.Now().Unix())

	// Create Stripe checkout session
	params := billing.CheckoutSessionParams{
		UserID:             userID,
		PlanID:             planID,
		OrderID:            orderID,
		ProductName:        plan.Name,
		ProductDescription: plan.Description,
		Amount:             plan.PriceCents,
		Currency:           plan.Currency,
	}

	// Create Stripe checkout session with circuit breaker protection and retry logic
	stripeBreaker := s.circuitBreakerManager.GetOrCreate("stripe", circuitbreaker.DefaultConfig())
	retryConfig := retry.DefaultConfig()
	logger := log.L(ctx)

	var session *billing.CheckoutSession
	err = stripeBreaker.Execute(ctx, func() error {
		return retry.Do(ctx, retryConfig, logger, func() error {
			var err error
			session, err = s.paymentProvider.CreateCheckoutSession(ctx, params)
			return err
		})
	})
	if err != nil {
		log.Error(ctx, "Failed to create Stripe checkout session", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create checkout session: %v", err)
	}

	// Create payment record
	payment := &domain.Payment{
		ID:              uuid.New(),
		Amount:          plan.PriceCents,
		Currency:        plan.Currency,
		Status:          string(domain.PaymentStatusPending),
		PaymentMethod:   "stripe",
		CustomerID:      userID,
		OrderID:         orderID,
		Description:     fmt.Sprintf("Payment for %s plan", plan.Name),
		StripeSessionID: &session.SessionID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		log.Error(ctx, "Failed to create payment record", zap.Error(err))
		// Don't fail the request if payment record creation fails
	}

	log.Info(ctx, "Checkout session created",
		zap.String("session_id", session.SessionID),
		zap.String("user_id", userID),
		zap.String("plan_id", planID),
		zap.String("order_id", orderID))

	return &CheckoutSessionResponse{
		Provider:    session.Provider,
		SessionID:   session.SessionID,
		RedirectURL: session.URL,
	}, nil
}

// PaymentSuccessWebhook handles payment success webhooks with idempotency
func (s *EnhancedPaymentService) PaymentSuccessWebhook(ctx context.Context, payload []byte, signature string) error {
	// Validate webhook signature
	if err := s.paymentProvider.ValidateWebhookSignature(payload, signature); err != nil {
		log.Error(ctx, "Invalid webhook signature", zap.Error(err))
		return status.Error(codes.Unauthenticated, "invalid webhook signature")
	}

	// Parse webhook event
	event, err := s.paymentProvider.ParseWebhookEvent(payload)
	if err != nil {
		log.Error(ctx, "Failed to parse webhook event", zap.Error(err))
		return status.Error(codes.InvalidArgument, "invalid webhook payload")
	}

	// Check for idempotency (prevent duplicate processing)
	existingEvent, err := s.webhookEventsRepo.GetByEventID(ctx, event.ID)
	if err == nil && existingEvent != nil && existingEvent.Processed {
		log.Info(ctx, "Webhook event already processed",
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
		return nil // Already processed, return success
	}

	// Insert webhook event for idempotency - CRITICAL: Fail if this fails
	if err := s.webhookEventsRepo.Insert(ctx, event.ID, event.Type, signature, payload); err != nil {
		log.Error(ctx, "Failed to insert webhook event for idempotency",
			zap.Error(err),
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
		return fmt.Errorf("failed to insert webhook event for idempotency tracking: %w", err)
	}

	// Process the webhook event
	if err := s.processWebhookEvent(ctx, event); err != nil {
		log.Error(ctx, "Failed to process webhook event",
			zap.Error(err),
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
		return status.Errorf(codes.Internal, "failed to process webhook: %v", err)
	}

	// Mark webhook event as processed
	if err := s.webhookEventsRepo.MarkProcessed(ctx, event.ID); err != nil {
		log.Error(ctx, "Failed to mark webhook event as processed", zap.Error(err))
		// Don't fail the request if marking as processed fails
	}

	return nil
}

// processWebhookEvent processes a webhook event
func (s *EnhancedPaymentService) processWebhookEvent(ctx context.Context, event *billing.WebhookEvent) error {
	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		return s.handleCheckoutSessionCompleted(ctx, event)
	case "payment_intent.succeeded":
		return s.handlePaymentIntentSucceeded(ctx, event)
	case "payment_intent.payment_failed":
		return s.handlePaymentIntentFailed(ctx, event)
	default:
		log.Info(ctx, "Unhandled webhook event type",
			zap.String("event_type", event.Type),
			zap.String("event_id", event.ID))
		return nil
	}
}

// handleCheckoutSessionCompleted handles checkout.session.completed event
func (s *EnhancedPaymentService) handleCheckoutSessionCompleted(ctx context.Context, event *billing.WebhookEvent) error {
	// Extract payment intent data
	paymentData, err := s.paymentProvider.ExtractPaymentIntentData(event)
	if err != nil {
		return fmt.Errorf("failed to extract payment intent data: %w", err)
	}

	// Update payment status - get payment by order ID first
	payment, err := s.paymentRepo.GetByOrderID(ctx, paymentData.OrderID)
	if err != nil {
		log.Error(ctx, "Failed to get payment by order ID", zap.Error(err))
	} else {
		if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, string(domain.PaymentStatusCompleted)); err != nil {
			log.Error(ctx, "Failed to update payment status", zap.Error(err))
			// Continue processing even if update fails
		}
	}

	// Get plan details
	plan, err := s.planRepo.GetByID(ctx, paymentData.PlanID)
	if err != nil {
		return fmt.Errorf("failed to get plan: %w", err)
	}

	// Create entitlements for all features in the plan
	for _, featureCode := range plan.FeatureCodes {
		// Calculate expiry time based on billing cycle
		var expiresAt *time.Time
		if plan.BillingCycle == "monthly" {
			t := time.Now().AddDate(0, 1, 0)
			expiresAt = &t
		} else if plan.BillingCycle == "yearly" {
			t := time.Now().AddDate(1, 0, 0)
			expiresAt = &t
		}

		entitlement := domain.Entitlement{
			ID:          uuid.New(),
			UserID:      paymentData.UserID,
			FeatureCode: featureCode,
			PlanID:      plan.ID,
			Status:      "active",
			GrantedAt:   time.Now(),
			ExpiresAt:   expiresAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Insert entitlement (with conflict handling)
		savedEntitlement, err := s.entitlementRepo.Insert(ctx, entitlement)
		if err != nil {
			log.Error(ctx, "Failed to create entitlement",
				zap.Error(err),
				zap.String("user_id", paymentData.UserID),
				zap.String("feature_code", featureCode))
			continue
		}

		// Publish entitlement.updated event (transactional outbox)
		if s.outboxRepo != nil {
			eventPayload := map[string]interface{}{
				"entitlement_id": savedEntitlement.ID,
				"user_id":        savedEntitlement.UserID,
				"feature_code":   savedEntitlement.FeatureCode,
				"plan_id":        savedEntitlement.PlanID,
				"status":         savedEntitlement.Status,
				"action":         "created",
			}
			payloadJSON, _ := json.Marshal(eventPayload)
			if err := s.outboxRepo.Insert(ctx, "entitlement.updated", payloadJSON); err != nil {
				log.Error(ctx, "Failed to insert outbox event", zap.Error(err))
			}
		}

		// Evict cache
		if s.cache != nil {
			s.cache.DeleteEntitlement(ctx, savedEntitlement.UserID, savedEntitlement.FeatureCode)
		}
	}

	log.Info(ctx, "Checkout session completed",
		zap.String("user_id", paymentData.UserID),
		zap.String("plan_id", paymentData.PlanID),
		zap.String("order_id", paymentData.OrderID))

	return nil
}

// handlePaymentIntentSucceeded handles payment_intent.succeeded event
func (s *EnhancedPaymentService) handlePaymentIntentSucceeded(ctx context.Context, event *billing.WebhookEvent) error {
	// Extract payment intent data
	paymentData, err := s.paymentProvider.ExtractPaymentIntentData(event)
	if err != nil {
		return fmt.Errorf("failed to extract payment intent data: %w", err)
	}

	// Update payment status - get payment by order ID first
	payment, err := s.paymentRepo.GetByOrderID(ctx, paymentData.OrderID)
	if err != nil {
		log.Error(ctx, "Failed to get payment by order ID", zap.Error(err))
	} else {
		if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, string(domain.PaymentStatusCompleted)); err != nil {
			log.Error(ctx, "Failed to update payment status", zap.Error(err))
		}
	}

	log.Info(ctx, "Payment intent succeeded",
		zap.String("payment_intent_id", paymentData.PaymentIntentID),
		zap.String("order_id", paymentData.OrderID))

	return nil
}

// handlePaymentIntentFailed handles payment_intent.payment_failed event
func (s *EnhancedPaymentService) handlePaymentIntentFailed(ctx context.Context, event *billing.WebhookEvent) error {
	// Extract payment intent data
	paymentData, err := s.paymentProvider.ExtractPaymentIntentData(event)
	if err != nil {
		return fmt.Errorf("failed to extract payment intent data: %w", err)
	}

	// Update payment status - get payment by order ID first
	payment, err := s.paymentRepo.GetByOrderID(ctx, paymentData.OrderID)
	if err != nil {
		log.Error(ctx, "Failed to get payment by order ID", zap.Error(err))
	} else {
		if err := s.paymentRepo.UpdateStatus(ctx, payment.ID, string(domain.PaymentStatusFailed)); err != nil {
			log.Error(ctx, "Failed to update payment status", zap.Error(err))
		}
	}

	log.Info(ctx, "Payment intent failed",
		zap.String("payment_intent_id", paymentData.PaymentIntentID),
		zap.String("order_id", paymentData.OrderID))

	return nil
}

// validateWebhookSignature validates a webhook signature using HMAC
func (s *EnhancedPaymentService) validateWebhookSignature(payload []byte, signature string) error {
	// Use payment provider's signature validation
	return s.paymentProvider.ValidateWebhookSignature(payload, signature)
}

// computeHMAC computes HMAC-SHA256 for webhook signature validation
func computeHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

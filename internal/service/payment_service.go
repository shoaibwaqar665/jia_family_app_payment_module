package service

import (
	"context"
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

// PaymentService provides payment business logic and implements PaymentServiceServer
type PaymentService struct {
	config                *config.Config
	paymentRepo           repository.PaymentRepository
	planRepo              repository.PlanRepository
	entitlementRepo       repository.EntitlementRepository
	cache                 *cache.Cache
	entitlementPublisher  events.EntitlementPublisher
	paymentProvider       billing.PaymentProvider
	transactionManager    repository.TransactionManager
	circuitBreakerManager *circuitbreaker.Manager
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	config *config.Config,
	paymentRepo repository.PaymentRepository,
	planRepo repository.PlanRepository,
	entitlementRepo repository.EntitlementRepository,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
	paymentProvider billing.PaymentProvider,
	transactionManager repository.TransactionManager,
	circuitBreakerManager *circuitbreaker.Manager,
) *PaymentService {
	return &PaymentService{
		config:                config,
		paymentRepo:           paymentRepo,
		planRepo:              planRepo,
		entitlementRepo:       entitlementRepo,
		cache:                 cache,
		entitlementPublisher:  entitlementPublisher,
		paymentProvider:       paymentProvider,
		transactionManager:    transactionManager,
		circuitBreakerManager: circuitBreakerManager,
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
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Publish payment created event
	if s.entitlementPublisher != nil {
		// Note: This would need a proper payment event publisher
		// For now, we'll log the event
		log.Info(ctx, "Payment created event",
			zap.String("payment_id", payment.ID.String()),
			zap.String("customer_id", payment.CustomerID),
			zap.String("order_id", payment.OrderID),
			zap.String("status", payment.Status))
	}

	// Process payment with payment processor
	if s.paymentProvider != nil {
		// Create a checkout session for the payment
		checkoutParams := billing.CheckoutSessionParams{
			UserID:             payment.CustomerID,
			PlanID:             "default_plan", // This should come from the request
			OrderID:            payment.OrderID,
			ProductName:        payment.Description,
			ProductDescription: fmt.Sprintf("Payment for %s", payment.Description),
			Amount:             payment.Amount,
			Currency:           payment.Currency,
		}

		// Use circuit breaker and retry for payment provider calls
		var session *billing.CheckoutSession
		err := s.executeWithResilience(ctx, "payment_provider", func() error {
			var err error
			session, err = s.paymentProvider.CreateCheckoutSession(ctx, checkoutParams)
			return err
		})

		if err != nil {
			log.Error(ctx, "Failed to create checkout session",
				zap.Error(err),
				zap.String("payment_id", payment.ID.String()))

			// Update payment status to failed
			payment.Status = string(domain.PaymentStatusFailed)
			if updateErr := s.paymentRepo.UpdateStatus(ctx, payment.ID, payment.Status); updateErr != nil {
				log.Error(ctx, "Failed to update payment status to failed",
					zap.Error(updateErr),
					zap.String("payment_id", payment.ID.String()))
			}

			return nil, fmt.Errorf("failed to create checkout session: %w", err)
		}

		// Update payment with Stripe session information
		payment.StripeSessionID = &session.SessionID
		payment.Status = string(domain.PaymentStatusPending)

		if err := s.paymentRepo.Update(ctx, payment); err != nil {
			log.Error(ctx, "Failed to update payment with session info",
				zap.Error(err),
				zap.String("payment_id", payment.ID.String()))
			return nil, fmt.Errorf("failed to update payment: %w", err)
		}

		log.Info(ctx, "Payment checkout session created successfully",
			zap.String("payment_id", payment.ID.String()),
			zap.String("session_id", session.SessionID),
			zap.String("checkout_url", session.URL))
	} else {
		// No payment provider configured - leave as pending
		log.Warn(ctx, "No payment provider configured - payment left in pending status",
			zap.String("payment_id", payment.ID.String()))
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

// GetPayment retrieves a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, id string) (*domain.PaymentResponse, error) {
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid payment ID format: %w", err)
	}

	payment, err := s.paymentRepo.GetByID(ctx, paymentID)
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
	// Parse payment ID
	paymentID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid payment ID format: %w", err)
	}

	// Validate status
	tempPayment := &domain.Payment{Status: status}
	if !tempPayment.IsValidStatus() {
		return domain.NewInvalidInputError("invalid payment status", fmt.Sprintf("status: %s", status))
	}

	// Update status
	if err := s.paymentRepo.UpdateStatus(ctx, paymentID, status); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Publish payment status updated event
	if s.entitlementPublisher != nil {
		log.Info(ctx, "Payment status updated event",
			zap.String("payment_id", id),
			zap.String("new_status", status))
	}

	return nil
}

// GetPaymentsByCustomer retrieves payments for a customer
func (s *PaymentService) GetPaymentsByCustomer(ctx context.Context, customerID string, limit, offset int) ([]*domain.PaymentResponse, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID format: %w", err)
	}

	payments, _, err := s.paymentRepo.GetByCustomerID(ctx, customerUUID, 100, 0)
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

// CheckEntitlement checks if a user has access to a specific feature
func (s *PaymentService) CheckEntitlement(ctx context.Context, userID, featureCode string) (*CheckEntitlementResponse, error) {
	// Use authenticated user_id from context if userID is empty
	if userID == "" {
		if contextUserID := extractUserIDFromContext(ctx); contextUserID != "" {
			userID = contextUserID
		} else {
			return nil, status.Error(codes.InvalidArgument, "user_id is required")
		}
	}

	// Validate input
	if featureCode == "" {
		return nil, status.Error(codes.InvalidArgument, "feature_code is required")
	}

	// Try Redis cache first
	if s.cache != nil {
		cachedEnt, found, err := s.cache.GetEntitlement(ctx, userID, featureCode)
		if err != nil {
			log.Warn(ctx, "Failed to get entitlement from cache",
				zap.Error(err), zap.String("user_id", userID), zap.String("feature_code", featureCode))
		} else if found {
			// Check if it's a negative cache result
			if isNegative, err := s.cache.IsEntitlementNotFound(ctx, userID, featureCode); err == nil && isNegative {
				return &CheckEntitlementResponse{
					Allowed:     false,
					Entitlement: nil,
				}, nil
			}

			// Validate cached entitlement is still active and not expired
			if isValidEntitlement(cachedEnt) {
				return &CheckEntitlementResponse{
					Allowed:     true,
					Entitlement: cachedEnt,
				}, nil
			}

			// Cached entitlement is invalid, evict from cache and fallback to repo
			s.cache.DeleteEntitlement(ctx, userID, featureCode)
		}
	}

	// Fallback to repository
	entitlement, found, err := s.entitlementRepo.Check(ctx, userID, featureCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check entitlement: %v", err)
	}

	// If user has a family_id, check for family entitlements
	if found && entitlement.FamilyID != nil {
		// User has family entitlement - this is already the merged result
		// The repository query already checks for family_id matches
		log.Info(ctx, "Found family entitlement",
			zap.String("user_id", userID),
			zap.String("family_id", *entitlement.FamilyID),
			zap.String("feature_code", featureCode))
	} else if !found && entitlement.FamilyID != nil {
		// User has family_id but no direct entitlement
		// Check if there are family entitlements
		familyEntitlements, err := s.entitlementRepo.ListByUser(ctx, userID)
		if err != nil {
			log.Warn(ctx, "Failed to list family entitlements",
				zap.Error(err),
				zap.String("user_id", userID))
		} else {
			// Find matching feature in family entitlements
			for _, fe := range familyEntitlements {
				if fe.FeatureCode == featureCode && fe.FamilyID != nil {
					entitlement = fe
					found = true
					log.Info(ctx, "Found family entitlement through family_id",
						zap.String("user_id", userID),
						zap.String("family_id", *fe.FamilyID),
						zap.String("feature_code", featureCode))
					break
				}
			}
		}
	}

	if !found {
		// Cache negative result
		if s.cache != nil {
			s.cache.SetEntitlementNotFound(ctx, userID, featureCode)
		}
		return &CheckEntitlementResponse{
			Allowed:     false,
			Entitlement: nil,
		}, nil
	}

	// Validate entitlement is active and not expired
	if !isValidEntitlement(&entitlement) {
		// Cache negative result for invalid entitlements
		if s.cache != nil {
			s.cache.SetEntitlementNotFound(ctx, userID, featureCode)
		}
		return &CheckEntitlementResponse{
			Allowed:     false,
			Entitlement: nil,
		}, nil
	}

	// Cache valid entitlement
	if s.cache != nil {
		s.cache.SetEntitlement(ctx, entitlement, 0) // Use default TTL
	}

	return &CheckEntitlementResponse{
		Allowed:     true,
		Entitlement: &entitlement,
	}, nil
}

// ListUserEntitlements lists all entitlements for a user
func (s *PaymentService) ListUserEntitlements(ctx context.Context, userID string) ([]*domain.Entitlement, error) {
	// Use authenticated user_id from context if userID is empty
	if userID == "" {
		if contextUserID := extractUserIDFromContext(ctx); contextUserID != "" {
			userID = contextUserID
		} else {
			return nil, status.Error(codes.InvalidArgument, "user_id is required")
		}
	}

	entitlements, err := s.entitlementRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user entitlements: %v", err)
	}

	// Convert to pointers for consistency
	result := make([]*domain.Entitlement, len(entitlements))
	for i := range entitlements {
		result[i] = &entitlements[i]
	}

	return result, nil
}

// CreateCheckoutSession creates a checkout session for a plan
func (s *PaymentService) CreateCheckoutSession(ctx context.Context, planID, userID string) (*CheckoutSessionResponse, error) {
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

	// Validate plan exists
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan: %v", err)
	}
	if plan.ID.String() == "" {
		return nil, status.Error(codes.NotFound, "plan not found")
	}

	// Generate order ID
	orderID := fmt.Sprintf("order_%s_%d", userID, time.Now().Unix())

	// Create checkout session using payment provider
	checkoutParams := billing.CheckoutSessionParams{
		UserID:             userID,
		PlanID:             planID,
		OrderID:            orderID,
		ProductName:        plan.Name,
		ProductDescription: plan.Description,
		Amount:             plan.PriceCents,
		Currency:           plan.Currency,
	}

	session, err := s.paymentProvider.CreateCheckoutSession(ctx, checkoutParams)
	if err != nil {
		log.Error(ctx, "Failed to create checkout session",
			zap.Error(err),
			zap.String("plan_id", planID),
			zap.String("user_id", userID))
		return nil, status.Errorf(codes.Internal, "failed to create checkout session: %v", err)
	}

	log.Info(ctx, "Created checkout session",
		zap.String("plan_id", planID),
		zap.String("user_id", userID),
		zap.String("session_id", session.SessionID),
		zap.String("provider", session.Provider))

	return &CheckoutSessionResponse{
		Provider:    session.Provider,
		SessionID:   session.SessionID,
		RedirectURL: session.URL,
	}, nil
}

// PaymentSuccessWebhook handles payment success webhooks
func (s *PaymentService) PaymentSuccessWebhook(ctx context.Context, payload []byte, signature string) error {
	// Validate signature using payment provider
	if err := s.paymentProvider.ValidateWebhookSignature(payload, signature); err != nil {
		log.Error(ctx, "Invalid webhook signature", zap.Error(err))
		return status.Error(codes.Unauthenticated, "invalid webhook signature")
	}

	// Parse webhook event using payment provider
	webhookEvent, err := s.paymentProvider.ParseWebhookEvent(payload)
	if err != nil {
		log.Error(ctx, "Failed to parse webhook event", zap.Error(err))
		return status.Error(codes.InvalidArgument, "invalid webhook payload")
	}

	// Extract payment intent data
	paymentData, err := s.paymentProvider.ExtractPaymentIntentData(webhookEvent)
	if err != nil {
		log.Error(ctx, "Failed to extract payment intent data", zap.Error(err))
		return status.Error(codes.InvalidArgument, "invalid payment intent data")
	}

	// Create entitlement within a transaction to ensure data integrity
	var savedEntitlement domain.Entitlement
	err = s.transactionManager.WithTransaction(ctx, func(tx repository.Transaction) error {
		// Create entitlement
		entitlement := domain.Entitlement{
			ID:          uuid.New(),
			UserID:      paymentData.UserID,
			FeatureCode: "premium_feature", // This should come from plan metadata
			PlanID:      uuid.MustParse(paymentData.PlanID),
			Status:      "active",
			GrantedAt:   time.Now(),
			ExpiresAt:   nil, // This should be calculated based on plan billing cycle
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		var err error
		savedEntitlement, err = tx.Entitlement().Insert(ctx, entitlement)
		if err != nil {
			return fmt.Errorf("failed to create entitlement: %w", err)
		}

		// Store webhook event for audit trail
		if err := tx.WebhookEvents().Insert(ctx, paymentData.SessionID, "payment.success", signature, payload); err != nil {
			log.Warn(ctx, "Failed to store webhook event", zap.Error(err))
			// Don't fail the transaction for webhook event storage failure
		}

		// Add outbox event for eventual consistency
		eventPayload := map[string]interface{}{
			"entitlement_id": savedEntitlement.ID.String(),
			"user_id":        savedEntitlement.UserID,
			"feature_code":   savedEntitlement.FeatureCode,
			"plan_id":        savedEntitlement.PlanID.String(),
			"status":         savedEntitlement.Status,
			"granted_at":     savedEntitlement.GrantedAt,
		}

		eventBytes, err := json.Marshal(eventPayload)
		if err != nil {
			log.Warn(ctx, "Failed to marshal entitlement event", zap.Error(err))
			// Don't fail the transaction for event marshaling failure
		} else {
			if err := tx.Outbox().Insert(ctx, "entitlement.created", eventBytes); err != nil {
				log.Warn(ctx, "Failed to add entitlement event to outbox", zap.Error(err))
				// Don't fail the transaction for outbox failure
			}
		}

		return nil
	})

	if err != nil {
		log.Error(ctx, "Failed to create entitlement in transaction", zap.Error(err))
		return status.Errorf(codes.Internal, "failed to create entitlement: %v", err)
	}

	// Publish entitlement.updated event (outside transaction)
	if s.entitlementPublisher != nil {
		if err := s.entitlementPublisher.PublishEntitlementUpdated(ctx, savedEntitlement, "created"); err != nil {
			log.Error(ctx, "Failed to publish entitlement.updated event", zap.Error(err))
		}
	}

	// Evict cache (outside transaction)
	if s.cache != nil {
		s.cache.DeleteEntitlement(ctx, savedEntitlement.UserID, savedEntitlement.FeatureCode)
	}

	log.Info(ctx, "Successfully processed payment webhook",
		zap.String("user_id", paymentData.UserID),
		zap.String("plan_id", paymentData.PlanID),
		zap.String("session_id", paymentData.SessionID))

	return nil
}

// Helper types for responses
type CheckEntitlementResponse struct {
	Allowed     bool                `json:"allowed"`
	Entitlement *domain.Entitlement `json:"entitlement,omitempty"`
}

type CheckoutSessionResponse struct {
	Provider    string `json:"provider"`
	SessionID   string `json:"session_id"`
	RedirectURL string `json:"redirect_url"`
}

type WebhookData struct {
	UserID      string     `json:"user_id"`
	FeatureCode string     `json:"feature_code"`
	PlanID      uuid.UUID  `json:"plan_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// Helper functions

// extractUserIDFromContext extracts user ID from context
func extractUserIDFromContext(ctx context.Context) string {
	// Extract user ID from context that was set by the auth interceptor
	if userID, ok := ctx.Value(log.UserIDKey).(string); ok && userID != "" {
		return userID
	}
	return ""
}

func isValidEntitlement(ent *domain.Entitlement) bool {
	if ent == nil {
		return false
	}

	// Check if status is active
	if ent.Status != "active" {
		return false
	}

	// Check if not expired
	if ent.ExpiresAt != nil && ent.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// executeWithResilience executes a function with circuit breaker and retry logic
func (s *PaymentService) executeWithResilience(ctx context.Context, serviceName string, fn func() error) error {
	if s.circuitBreakerManager == nil {
		// No circuit breaker available, execute directly
		return fn()
	}

	// Get or create circuit breaker for the service
	cb := s.circuitBreakerManager.GetOrCreate(serviceName, circuitbreaker.DefaultConfig())

	// Execute with circuit breaker
	return cb.Execute(ctx, func() error {
		// Use retry logic within the circuit breaker
		retryConfig := retry.DefaultConfig()
		logger := log.L(ctx)

		return retry.Do(ctx, retryConfig, logger, fn)
	})
}

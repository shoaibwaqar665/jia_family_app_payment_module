package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/cache"
	"github.com/jia-app/paymentservice/internal/config"
	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/events"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/repository"
)

// PaymentService provides payment business logic and implements PaymentServiceServer
type PaymentService struct {
	config               *config.Config
	paymentRepo          repository.PaymentRepository
	planRepo             repository.PlanRepository
	entitlementRepo      repository.EntitlementRepository
	cache                *cache.Cache
	entitlementPublisher events.EntitlementPublisher
}

// NewPaymentService creates a new payment service
func NewPaymentService(
	config *config.Config,
	paymentRepo repository.PaymentRepository,
	planRepo repository.PlanRepository,
	entitlementRepo repository.EntitlementRepository,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
) *PaymentService {
	return &PaymentService{
		config:               config,
		paymentRepo:          paymentRepo,
		planRepo:             planRepo,
		entitlementRepo:      entitlementRepo,
		cache:                cache,
		entitlementPublisher: entitlementPublisher,
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
	payment, err := s.paymentRepo.GetByID(ctx, id)
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
	if err := s.paymentRepo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// TODO: Publish payment status updated event

	return nil
}

// GetPaymentsByCustomer retrieves payments for a customer
func (s *PaymentService) GetPaymentsByCustomer(ctx context.Context, customerID string, limit, offset int) ([]*domain.PaymentResponse, error) {
	payments, err := s.paymentRepo.GetByCustomerID(ctx, customerID, limit, offset)
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

	// Generate placeholder session
	sessionID := fmt.Sprintf("sess_%s", uuid.New().String()[:8])
	redirectURL := fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID)

	log.Info(ctx, "TODO: integrate with actual payment provider",
		zap.String("plan_id", planID),
		zap.String("user_id", userID),
		zap.String("provider", "stripe"),
		zap.String("stripe_secret_key", s.config.Billing.StripeSecret[:12]+"..."),
		zap.String("stripe_publishable_key", s.config.Billing.StripePublishable[:12]+"..."))

	return &CheckoutSessionResponse{
		Provider:    "stripe",
		SessionID:   sessionID,
		RedirectURL: redirectURL,
	}, nil
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

	// Upsert entitlement
	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      webhookData.UserID,
		FeatureCode: webhookData.FeatureCode,
		PlanID:      webhookData.PlanID,
		Status:      "active",
		GrantedAt:   time.Now(),
		ExpiresAt:   webhookData.ExpiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	savedEntitlement, err := s.entitlementRepo.Insert(ctx, entitlement)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create entitlement: %v", err)
	}

	// Publish entitlement.updated event
	if s.entitlementPublisher != nil {
		if err := s.entitlementPublisher.PublishEntitlementUpdated(ctx, savedEntitlement, "created"); err != nil {
			log.Error(ctx, "Failed to publish entitlement.updated event", zap.Error(err))
		}
	}

	// Evict cache
	if s.cache != nil {
		s.cache.DeleteEntitlement(ctx, savedEntitlement.UserID, savedEntitlement.FeatureCode)
	}

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
func extractUserIDFromContext(ctx context.Context) string {
	if userID := ctx.Value(log.UserIDKey); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
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

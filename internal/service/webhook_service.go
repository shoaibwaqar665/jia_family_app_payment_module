package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/repository"
)

// WebhookService handles webhook processing with security and idempotency
type WebhookService struct {
	webhookRepo     repository.WebhookEventsRepository
	paymentRepo     repository.PaymentRepository
	entitlementRepo repository.EntitlementRepository
	planRepo        repository.PlanRepository
	secret          string
	logger          *zap.Logger
}

// NewWebhookService creates a new webhook service
func NewWebhookService(
	webhookRepo repository.WebhookEventsRepository,
	paymentRepo repository.PaymentRepository,
	entitlementRepo repository.EntitlementRepository,
	planRepo repository.PlanRepository,
	secret string,
	logger *zap.Logger,
) *WebhookService {
	return &WebhookService{
		webhookRepo:     webhookRepo,
		paymentRepo:     paymentRepo,
		entitlementRepo: entitlementRepo,
		planRepo:        planRepo,
		secret:          secret,
		logger:          logger,
	}
}

// WebhookRequest represents a webhook request
type WebhookRequest struct {
	EventID   string
	EventType string
	Payload   []byte
	Signature string
	Timestamp int64 // Unix timestamp
}

// WebhookResponse represents the response from webhook processing
type WebhookResponse struct {
	Processed bool
	Message   string
}

// ProcessWebhook processes a webhook with security validation and idempotency
func (s *WebhookService) ProcessWebhook(ctx context.Context, req *WebhookRequest) (*WebhookResponse, error) {
	// 1. Validate timestamp (5-minute window)
	if err := s.validateTimestamp(req.Timestamp); err != nil {
		log.Warn(ctx, "Webhook timestamp validation failed",
			zap.Error(err),
			zap.String("event_id", req.EventID),
			zap.Int64("timestamp", req.Timestamp))
		return &WebhookResponse{
			Processed: false,
			Message:   "Timestamp validation failed",
		}, nil
	}

	// 2. Validate signature
	if err := s.validateSignature(req.Payload, req.Signature); err != nil {
		log.Warn(ctx, "Webhook signature validation failed",
			zap.Error(err),
			zap.String("event_id", req.EventID))
		return &WebhookResponse{
			Processed: false,
			Message:   "Signature validation failed",
		}, nil
	}

	// 3. Check for idempotency
	existingEvent, err := s.webhookRepo.GetByEventID(ctx, req.EventID)
	if err == nil && existingEvent != nil {
		if existingEvent.Processed {
			log.Info(ctx, "Webhook already processed",
				zap.String("event_id", req.EventID),
				zap.Time("processed_at", *existingEvent.ProcessedAt))
			return &WebhookResponse{
				Processed: true,
				Message:   "Event already processed",
			}, nil
		}
		// Event exists but not processed - this is unexpected, log and continue
		log.Warn(ctx, "Webhook event exists but not processed",
			zap.String("event_id", req.EventID),
			zap.Time("created_at", existingEvent.CreatedAt))
	}

	// 4. Store webhook event for idempotency
	if err := s.webhookRepo.Insert(ctx, req.EventID, req.EventType, req.Signature, req.Payload); err != nil {
		log.Error(ctx, "Failed to store webhook event",
			zap.Error(err),
			zap.String("event_id", req.EventID))
		return nil, fmt.Errorf("failed to store webhook event: %w", err)
	}

	// 5. Process the webhook (this would be implemented based on event type)
	if err := s.processWebhookEvent(ctx, req); err != nil {
		log.Error(ctx, "Failed to process webhook event",
			zap.Error(err),
			zap.String("event_id", req.EventID),
			zap.String("event_type", req.EventType))
		return nil, fmt.Errorf("failed to process webhook event: %w", err)
	}

	// 6. Mark as processed
	if err := s.webhookRepo.MarkProcessed(ctx, req.EventID); err != nil {
		log.Error(ctx, "Failed to mark webhook event as processed",
			zap.Error(err),
			zap.String("event_id", req.EventID))
		// Don't return error here as the event was processed successfully
	}

	log.Info(ctx, "Webhook processed successfully",
		zap.String("event_id", req.EventID),
		zap.String("event_type", req.EventType))

	return &WebhookResponse{
		Processed: true,
		Message:   "Webhook processed successfully",
	}, nil
}

// validateTimestamp validates that the webhook timestamp is within the allowed window
func (s *WebhookService) validateTimestamp(timestamp int64) error {
	if timestamp == 0 {
		return fmt.Errorf("timestamp is required")
	}

	now := time.Now().Unix()
	diff := now - timestamp

	// Allow 5-minute window (300 seconds)
	if diff < 0 {
		return fmt.Errorf("timestamp is in the future")
	}
	if diff > 300 {
		return fmt.Errorf("timestamp is too old (older than 5 minutes)")
	}

	return nil
}

// validateSignature validates the webhook signature using HMAC-SHA256
func (s *WebhookService) validateSignature(payload []byte, signature string) error {
	if signature == "" {
		return fmt.Errorf("signature is required")
	}

	if s.secret == "" {
		return fmt.Errorf("webhook secret is not configured")
	}

	// Create HMAC signature
	h := hmac.New(sha256.New, []byte(s.secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("signature validation failed")
	}

	return nil
}

// processWebhookEvent processes the actual webhook event based on its type
func (s *WebhookService) processWebhookEvent(ctx context.Context, req *WebhookRequest) error {
	switch req.EventType {
	case "payment.succeeded":
		return s.processPaymentSucceeded(ctx, req.Payload)
	case "payment.failed":
		return s.processPaymentFailed(ctx, req.Payload)
	case "subscription.created":
		return s.processSubscriptionCreated(ctx, req.Payload)
	case "subscription.updated":
		return s.processSubscriptionUpdated(ctx, req.Payload)
	case "subscription.deleted":
		return s.processSubscriptionDeleted(ctx, req.Payload)
	default:
		log.Warn(ctx, "Unknown webhook event type",
			zap.String("event_type", req.EventType),
			zap.String("event_id", req.EventID))
		return fmt.Errorf("unknown event type: %s", req.EventType)
	}
}

// processPaymentSucceeded processes a payment succeeded webhook
func (s *WebhookService) processPaymentSucceeded(ctx context.Context, payload []byte) error {
	// Parse Stripe payment succeeded event
	var stripeEvent struct {
		Data struct {
			Object struct {
				ID              string                 `json:"id"`
				Amount          int64                  `json:"amount"`
				Currency        string                 `json:"currency"`
				Status          string                 `json:"status"`
				PaymentMethod   string                 `json:"payment_method"`
				Customer        string                 `json:"customer"`
				Description     string                 `json:"description"`
				Metadata        map[string]interface{} `json:"metadata"`
				PaymentIntentID string                 `json:"payment_intent"`
				SessionID       string                 `json:"session_id"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &stripeEvent); err != nil {
		return fmt.Errorf("failed to parse payment succeeded payload: %w", err)
	}

	// Extract payment data
	paymentData := stripeEvent.Data.Object

	// Get user ID and plan ID from metadata
	userID, ok := paymentData.Metadata["user_id"].(string)
	if !ok {
		return fmt.Errorf("user_id not found in payment metadata")
	}

	planIDStr, ok := paymentData.Metadata["plan_id"].(string)
	if !ok {
		return fmt.Errorf("plan_id not found in payment metadata")
	}

	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		return fmt.Errorf("invalid plan_id format: %w", err)
	}

	// Get order ID from metadata
	orderID, ok := paymentData.Metadata["order_id"].(string)
	if !ok {
		orderID = uuid.New().String()
	}

	// Create payment record
	payment := &domain.Payment{
		ID:                    uuid.New(),
		OrderID:               orderID,
		CustomerID:            userID,
		Amount:                paymentData.Amount,
		Currency:              paymentData.Currency,
		Status:                "completed",
		PaymentMethod:         paymentData.PaymentMethod,
		StripePaymentIntentID: &paymentData.PaymentIntentID,
		StripeSessionID:       &paymentData.SessionID,
		Description:           paymentData.Description,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Store payment in database
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		s.logger.Error("Failed to create payment record",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("user_id", userID))
		return fmt.Errorf("failed to create payment record: %w", err)
	}

	// Get plan details to determine features
	plan, err := s.planRepo.GetByID(ctx, planIDStr)
	if err != nil {
		s.logger.Error("Failed to get plan details",
			zap.Error(err),
			zap.String("plan_id", planIDStr))
		return fmt.Errorf("failed to get plan details: %w", err)
	}

	// Calculate expiry date based on plan billing cycle
	var expiresAt *time.Time
	if plan.BillingCycle == "monthly" {
		expiry := time.Now().AddDate(0, 1, 0)
		expiresAt = &expiry
	} else if plan.BillingCycle == "yearly" {
		expiry := time.Now().AddDate(1, 0, 0)
		expiresAt = &expiry
	}

	// Create entitlement for the user
	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: plan.FeatureCodes[0], // Use first feature code from plan
		PlanID:      planID,
		Status:      "active",
		GrantedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store entitlement in database
	if _, err := s.entitlementRepo.Insert(ctx, entitlement); err != nil {
		s.logger.Error("Failed to create entitlement",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("plan_id", planIDStr))
		return fmt.Errorf("failed to create entitlement: %w", err)
	}

	s.logger.Info("Payment succeeded - entitlement created",
		zap.String("user_id", userID),
		zap.String("plan_id", planIDStr),
		zap.String("payment_id", payment.ID.String()),
		zap.String("entitlement_id", entitlement.ID.String()),
		zap.String("payment_intent_id", paymentData.PaymentIntentID),
		zap.String("amount", fmt.Sprintf("%d", paymentData.Amount)),
		zap.String("currency", paymentData.Currency))

	return nil
}

// processPaymentFailed processes a payment failed webhook
func (s *WebhookService) processPaymentFailed(ctx context.Context, payload []byte) error {
	// Parse Stripe payment failed event
	var stripeEvent struct {
		Data struct {
			Object struct {
				ID              string                 `json:"id"`
				Amount          int64                  `json:"amount"`
				Currency        string                 `json:"currency"`
				Status          string                 `json:"status"`
				PaymentMethod   string                 `json:"payment_method"`
				Customer        string                 `json:"customer"`
				Description     string                 `json:"description"`
				Metadata        map[string]interface{} `json:"metadata"`
				PaymentIntentID string                 `json:"payment_intent"`
				SessionID       string                 `json:"session_id"`
				FailureCode     string                 `json:"failure_code"`
				FailureMessage  string                 `json:"failure_message"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &stripeEvent); err != nil {
		return fmt.Errorf("failed to parse payment failed payload: %w", err)
	}

	// Extract payment data
	paymentData := stripeEvent.Data.Object

	// Get user ID from metadata
	userID, ok := paymentData.Metadata["user_id"].(string)
	if !ok {
		return fmt.Errorf("user_id not found in payment metadata")
	}

	// Get order ID from metadata
	orderID, ok := paymentData.Metadata["order_id"].(string)
	if !ok {
		orderID = uuid.New().String()
	}

	// Get plan ID from metadata
	planIDStr, ok := paymentData.Metadata["plan_id"].(string)
	if !ok {
		return fmt.Errorf("plan_id not found in payment metadata")
	}

	// Create payment record with failed status
	payment := &domain.Payment{
		ID:                    uuid.New(),
		OrderID:               orderID,
		CustomerID:            userID,
		Amount:                paymentData.Amount,
		Currency:              paymentData.Currency,
		Status:                "failed",
		PaymentMethod:         paymentData.PaymentMethod,
		StripePaymentIntentID: &paymentData.PaymentIntentID,
		StripeSessionID:       &paymentData.SessionID,
		Description:           paymentData.Description,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Store failed payment in database
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		s.logger.Error("Failed to create failed payment record",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("user_id", userID))
		return fmt.Errorf("failed to create payment record: %w", err)
	}

	// Log payment failure details
	s.logger.Warn("Payment failed",
		zap.String("user_id", userID),
		zap.String("payment_id", payment.ID.String()),
		zap.String("order_id", orderID),
		zap.String("plan_id", planIDStr),
		zap.String("payment_intent_id", paymentData.PaymentIntentID),
		zap.String("failure_code", paymentData.FailureCode),
		zap.String("failure_message", paymentData.FailureMessage),
		zap.String("amount", fmt.Sprintf("%d", paymentData.Amount)),
		zap.String("currency", paymentData.Currency))

	// In a real implementation, you might:
	// 1. Send notification to user about payment failure
	// 2. Trigger retry logic for failed payments
	// 3. Update subscription status if applicable
	// 4. Send alerts to support team for investigation

	return nil
}

// processSubscriptionCreated processes a subscription created webhook
func (s *WebhookService) processSubscriptionCreated(ctx context.Context, payload []byte) error {
	// Parse Stripe subscription created event
	var stripeEvent struct {
		Data struct {
			Object struct {
				ID                 string                 `json:"id"`
				Customer           string                 `json:"customer"`
				Status             string                 `json:"status"`
				CurrentPeriodStart int64                  `json:"current_period_start"`
				CurrentPeriodEnd   int64                  `json:"current_period_end"`
				Metadata           map[string]interface{} `json:"metadata"`
				Items              struct {
					Data []struct {
						Price struct {
							ID string `json:"id"`
						} `json:"price"`
					} `json:"data"`
				} `json:"items"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &stripeEvent); err != nil {
		return fmt.Errorf("failed to parse subscription created payload: %w", err)
	}

	// Extract subscription data
	subscriptionData := stripeEvent.Data.Object

	// Get user ID from metadata
	userID, ok := subscriptionData.Metadata["user_id"].(string)
	if !ok {
		return fmt.Errorf("user_id not found in subscription metadata")
	}

	// Get plan ID from price ID (assuming price ID maps to plan ID)
	var planID string
	if len(subscriptionData.Items.Data) > 0 {
		planID = subscriptionData.Items.Data[0].Price.ID
	}

	// Calculate expiry date
	expiresAt := time.Unix(subscriptionData.CurrentPeriodEnd, 0)

	// Create entitlement for the user
	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: "premium_feature", // This should come from plan metadata
		PlanID:      uuid.MustParse(planID),
		Status:      "active",
		GrantedAt:   time.Now(),
		ExpiresAt:   &expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	log.Info(ctx, "Subscription created - entitlement created",
		zap.String("user_id", userID),
		zap.String("subscription_id", subscriptionData.ID),
		zap.String("plan_id", planID),
		zap.String("status", subscriptionData.Status),
		zap.Time("expires_at", expiresAt),
		zap.String("entitlement_id", entitlement.ID.String()))

	return nil
}

// processSubscriptionUpdated processes a subscription updated webhook
func (s *WebhookService) processSubscriptionUpdated(ctx context.Context, payload []byte) error {
	// Parse Stripe subscription updated event
	var stripeEvent struct {
		Data struct {
			Object struct {
				ID                 string                 `json:"id"`
				Customer           string                 `json:"customer"`
				Status             string                 `json:"status"`
				CurrentPeriodStart int64                  `json:"current_period_start"`
				CurrentPeriodEnd   int64                  `json:"current_period_end"`
				Metadata           map[string]interface{} `json:"metadata"`
				Items              struct {
					Data []struct {
						Price struct {
							ID string `json:"id"`
						} `json:"price"`
					} `json:"data"`
				} `json:"items"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &stripeEvent); err != nil {
		return fmt.Errorf("failed to parse subscription updated payload: %w", err)
	}

	// Extract subscription data
	subscriptionData := stripeEvent.Data.Object

	// Get user ID from metadata
	userID, ok := subscriptionData.Metadata["user_id"].(string)
	if !ok {
		return fmt.Errorf("user_id not found in subscription metadata")
	}

	// Get plan ID from price ID
	var planID string
	if len(subscriptionData.Items.Data) > 0 {
		planID = subscriptionData.Items.Data[0].Price.ID
	}

	// Calculate new expiry date
	expiresAt := time.Unix(subscriptionData.CurrentPeriodEnd, 0)

	log.Info(ctx, "Subscription updated",
		zap.String("user_id", userID),
		zap.String("subscription_id", subscriptionData.ID),
		zap.String("plan_id", planID),
		zap.String("status", subscriptionData.Status),
		zap.Time("expires_at", expiresAt))

	// In a real implementation, you would:
	// 1. Update existing entitlement expiry date
	// 2. Handle plan changes
	// 3. Update subscription status
	// 4. Send notifications if needed

	return nil
}

// processSubscriptionDeleted processes a subscription deleted webhook
func (s *WebhookService) processSubscriptionDeleted(ctx context.Context, payload []byte) error {
	// Parse Stripe subscription deleted event
	var stripeEvent struct {
		Data struct {
			Object struct {
				ID         string                 `json:"id"`
				Customer   string                 `json:"customer"`
				Status     string                 `json:"status"`
				Metadata   map[string]interface{} `json:"metadata"`
				CanceledAt int64                  `json:"canceled_at"`
				EndedAt    int64                  `json:"ended_at"`
			} `json:"object"`
		} `json:"data"`
	}

	if err := json.Unmarshal(payload, &stripeEvent); err != nil {
		return fmt.Errorf("failed to parse subscription deleted payload: %w", err)
	}

	// Extract subscription data
	subscriptionData := stripeEvent.Data.Object

	// Get user ID from metadata
	userID, ok := subscriptionData.Metadata["user_id"].(string)
	if !ok {
		return fmt.Errorf("user_id not found in subscription metadata")
	}

	// Calculate cancellation/end date
	var endTime time.Time
	if subscriptionData.EndedAt > 0 {
		endTime = time.Unix(subscriptionData.EndedAt, 0)
	} else if subscriptionData.CanceledAt > 0 {
		endTime = time.Unix(subscriptionData.CanceledAt, 0)
	} else {
		endTime = time.Now()
	}

	log.Info(ctx, "Subscription deleted",
		zap.String("user_id", userID),
		zap.String("subscription_id", subscriptionData.ID),
		zap.String("status", subscriptionData.Status),
		zap.Time("end_time", endTime))

	// In a real implementation, you would:
	// 1. Update entitlement status to expired/cancelled
	// 2. Set entitlement expiry date to cancellation date
	// 3. Send cancellation confirmation to user
	// 4. Trigger cleanup processes

	return nil
}

// CreateWebhookRequestFromStripe creates a webhook request from Stripe webhook data
func CreateWebhookRequestFromStripe(payload []byte, signature string, timestamp string) (*WebhookRequest, error) {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	// Parse Stripe webhook payload to extract event ID and type
	var stripeEvent struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(payload, &stripeEvent); err != nil {
		return nil, fmt.Errorf("failed to parse Stripe webhook payload: %w", err)
	}

	// Validate that we have the required fields
	if stripeEvent.ID == "" {
		return nil, fmt.Errorf("missing event ID in Stripe webhook payload")
	}
	if stripeEvent.Type == "" {
		return nil, fmt.Errorf("missing event type in Stripe webhook payload")
	}

	return &WebhookRequest{
		EventID:   stripeEvent.ID,
		EventType: stripeEvent.Type,
		Payload:   payload,
		Signature: signature,
		Timestamp: ts,
	}, nil
}

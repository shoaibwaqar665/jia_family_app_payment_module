package stripebp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/billing"
	"github.com/jia-app/paymentservice/internal/shared/circuitbreaker"
)

// Adapter implements the billing.Provider interface for Stripe
type Adapter struct {
	secretKey      string
	publishableKey string
	logger         *zap.Logger
	circuitBreaker *circuitbreaker.CircuitBreaker
}

// NewAdapter creates a new Stripe billing adapter
func NewAdapter(secretKey, publishableKey string, logger *zap.Logger) *Adapter {
	return &Adapter{
		secretKey:      secretKey,
		publishableKey: publishableKey,
		logger:         logger,
		circuitBreaker: circuitbreaker.GetOrCreateGlobal("stripe", circuitbreaker.StripeConfig),
	}
}

// CreateCheckoutSession creates a Stripe checkout session
func (a *Adapter) CreateCheckoutSession(ctx context.Context, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error) {
	var result *billing.CreateCheckoutSessionResponse
	var err error

	_, err = a.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		// Set Stripe API key
		stripe.Key = a.secretKey

		// Create line items
		lineItems := []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(req.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String("Subscription Plan"),
						Description: stripe.String(fmt.Sprintf("Plan ID: %s", req.PlanID.String())),
					},
					UnitAmount: stripe.Int64(int64(req.BasePrice * 100)), // Convert dollars to cents for Stripe
				},
				Quantity: stripe.Int64(1),
			},
		}

		// Prepare metadata
		metadata := map[string]string{
			"user_id":    req.UserID,
			"plan_id":    req.PlanID.String(),
			"base_price": fmt.Sprintf("%.2f", req.BasePrice), // Store as dollars in metadata
			"currency":   req.Currency,
		}

		if req.FamilyID != nil {
			metadata["family_id"] = *req.FamilyID
		}

		if req.CountryCode != "" {
			metadata["country_code"] = req.CountryCode
		}

		// Create checkout session parameters
		params := &stripe.CheckoutSessionParams{
			PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
			LineItems:          lineItems,
			Mode:               stripe.String(string(stripe.CheckoutSessionModePayment)),
			SuccessURL:         stripe.String(req.SuccessURL),
			CancelURL:          stripe.String(req.CancelURL),
			Metadata:           metadata,
			ExpiresAt:          stripe.Int64(time.Now().Add(24 * time.Hour).Unix()),
		}

		// Create the session
		session, err := session.New(params)
		if err != nil {
			a.logger.Error("Failed to create Stripe checkout session",
				zap.Error(err),
				zap.String("plan_id", req.PlanID.String()),
				zap.String("user_id", req.UserID))
			return nil, fmt.Errorf("failed to create checkout session: %w", err)
		}

		a.logger.Info("Created Stripe checkout session",
			zap.String("session_id", session.ID),
			zap.String("plan_id", req.PlanID.String()),
			zap.String("user_id", req.UserID),
			zap.String("family_id", getStringValue(req.FamilyID)),
			zap.String("success_url", req.SuccessURL),
			zap.String("cancel_url", req.CancelURL),
			zap.String("checkout_url", session.URL))

		// Convert to our response format
		result = &billing.CreateCheckoutSessionResponse{
			SessionID: session.ID,
			URL:       session.URL,
			ExpiresAt: time.Unix(session.ExpiresAt, 0),
		}

		return result, nil
	})

	return result, err
}

// GetSession retrieves a Stripe checkout session
func (a *Adapter) GetSession(ctx context.Context, sessionID string) (*billing.Session, error) {
	// TODO: Implement actual Stripe session retrieval
	// For now, return a mock response

	a.logger.Info("Retrieving Stripe checkout session",
		zap.String("session_id", sessionID))

	// TODO: Call Stripe API to retrieve actual session
	// This would involve:
	// 1. Calling stripe.CheckoutSession.Retrieve(sessionID)
	// 2. Converting Stripe session to our Session type

	return &billing.Session{
		ID:        sessionID,
		Status:    string(billing.SessionStatusOpen),
		URL:       fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// CancelSession cancels a Stripe checkout session
func (a *Adapter) CancelSession(ctx context.Context, sessionID string) error {
	// TODO: Implement actual Stripe session cancellation
	// For now, just log the action

	a.logger.Info("Cancelling Stripe checkout session",
		zap.String("session_id", sessionID))

	// TODO: Call Stripe API to cancel actual session
	// This would involve:
	// 1. Calling stripe.CheckoutSession.Cancel(sessionID)
	// 2. Handling any errors appropriately

	return nil
}

// ValidateWebhook validates a Stripe webhook signature
func (a *Adapter) ValidateWebhook(ctx context.Context, payload []byte, signature string) error {
	// TODO: Implement actual Stripe webhook signature validation
	// For now, just check that signature is not empty

	if signature == "" {
		return fmt.Errorf("missing webhook signature")
	}

	a.logger.Debug("Validating Stripe webhook signature",
		zap.String("signature", signature),
		zap.Int("payload_size", len(payload)))

	// TODO: Implement actual Stripe webhook signature validation
	// This would involve:
	// 1. Extracting timestamp and signatures from the signature header
	// 2. Computing HMAC-SHA256 of the payload with the webhook secret
	// 3. Comparing with the provided signature
	// 4. Checking timestamp to prevent replay attacks

	return nil
}

// ParseWebhook parses a Stripe webhook payload
func (a *Adapter) ParseWebhook(ctx context.Context, payload []byte) (*billing.WebhookResult, error) {
	// First try to parse as a custom webhook payload (for POC)
	var customPayload map[string]interface{}
	if err := json.Unmarshal(payload, &customPayload); err == nil {
		// Check if this is a custom webhook payload
		if _, ok := customPayload["session_id"].(string); ok {
			return a.handleCustomWebhookPayload(customPayload)
		}
	}

	// Parse the webhook event as Stripe event
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		a.logger.Error("Failed to parse webhook payload", zap.Error(err))
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	a.logger.Info("Processing webhook event",
		zap.String("event_type", string(event.Type)),
		zap.String("event_id", event.ID))

	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		return a.handleCheckoutSessionCompleted(event)
	case "payment_intent.succeeded":
		return a.handlePaymentSucceeded(event)
	case "payment_intent.payment_failed":
		return a.handlePaymentFailed(event)
	default:
		a.logger.Info("Unhandled webhook event type", zap.String("event_type", string(event.Type)))
		return nil, fmt.Errorf("unhandled event type: %s", event.Type)
	}
}

// handleCustomWebhookPayload handles custom webhook payloads from POC
func (a *Adapter) handleCustomWebhookPayload(payload map[string]interface{}) (*billing.WebhookResult, error) {
	sessionID, _ := payload["session_id"].(string)
	userID, _ := payload["user_id"].(string)
	planIDStr, _ := payload["plan_id"].(string)
	amount, _ := payload["amount"].(float64)
	currency, _ := payload["currency"].(string)
	status, _ := payload["status"].(string)

	if userID == "" || planIDStr == "" {
		return nil, fmt.Errorf("missing required fields in custom webhook payload")
	}

	// Handle both UUID and string plan IDs
	var planID uuid.UUID
	if parsedUUID, err := uuid.Parse(planIDStr); err == nil {
		// It's a valid UUID
		planID = parsedUUID
	} else {
		// It's a string plan ID, generate a deterministic UUID
		planID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(planIDStr))
	}

	if currency == "" {
		currency = "USD"
	}

	result := &billing.WebhookResult{
		EventType:    string(billing.WebhookEventTypeCheckoutSessionCompleted),
		SessionID:    sessionID,
		UserID:       userID,
		FeatureCode:  "", // Will be determined from plan
		PlanID:       planID,
		PlanIDString: planIDStr, // Store original plan ID string
		Amount:       amount,
		Currency:     currency,
		Status:       status,
		ExpiresAt:    nil, // Lifetime for this POC
		Metadata:     payload,
	}

	// Handle optional family ID
	if familyID, ok := payload["metadata"].(map[string]interface{}); ok {
		if fid, ok := familyID["family_id"].(string); ok && fid != "" {
			result.FamilyID = &fid
		}
	}

	a.logger.Info("Processed custom webhook payload",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("plan_id", planIDStr),
		zap.Float64("amount", amount),
		zap.String("currency", currency),
		zap.String("status", status))

	return result, nil
}

// handleCheckoutSessionCompleted handles checkout.session.completed events
func (a *Adapter) handleCheckoutSessionCompleted(event stripe.Event) (*billing.WebhookResult, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, fmt.Errorf("failed to parse checkout session: %w", err)
	}

	// Extract metadata
	userID := session.Metadata["user_id"]
	planIDStr := session.Metadata["plan_id"]
	familyID := session.Metadata["family_id"]
	basePriceStr := session.Metadata["base_price"]
	currency := session.Metadata["currency"]

	if userID == "" || planIDStr == "" {
		return nil, fmt.Errorf("missing required metadata: user_id or plan_id")
	}

	// Handle both UUID and string plan IDs
	var planID uuid.UUID
	if parsedUUID, err := uuid.Parse(planIDStr); err == nil {
		// It's a valid UUID
		planID = parsedUUID
	} else {
		// It's a string plan ID, generate a deterministic UUID
		planID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(planIDStr))
	}

	basePrice := float64(19.99) // Default in dollars
	if basePriceStr != "" {
		if p, err := fmt.Sscanf(basePriceStr, "%f", &basePrice); err != nil || p != 1 {
			a.logger.Warn("Invalid base_price in metadata, using default", zap.String("base_price", basePriceStr))
		}
	}

	if currency == "" {
		currency = "USD"
	}

	// Determine feature code based on plan
	featureCode := a.getFeatureCodeForPlan(planIDStr)

	result := &billing.WebhookResult{
		EventType:    string(billing.WebhookEventTypeCheckoutSessionCompleted),
		SessionID:    session.ID,
		UserID:       userID,
		FeatureCode:  featureCode,
		PlanID:       planID,
		PlanIDString: planIDStr, // Store original plan ID string
		Amount:       basePrice,
		Currency:     currency,
		Status:       "completed",
		ExpiresAt:    nil, // Lifetime for this POC
		Metadata: map[string]interface{}{
			"stripe_session_id": session.ID,
			"payment_status":    session.PaymentStatus,
		},
	}

	if familyID != "" {
		result.FamilyID = &familyID
	}

	a.logger.Info("Processed checkout session completed",
		zap.String("session_id", session.ID),
		zap.String("user_id", userID),
		zap.String("plan_id", planIDStr),
		zap.String("feature_code", featureCode))

	return result, nil
}

// handlePaymentSucceeded handles payment_intent.succeeded events
func (a *Adapter) handlePaymentSucceeded(event stripe.Event) (*billing.WebhookResult, error) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}

	// For POC, we'll create a basic result
	// In production, you'd want to link this to your checkout session
	result := &billing.WebhookResult{
		EventType:   string(billing.WebhookEventTypePaymentSucceeded),
		SessionID:   paymentIntent.ID, // Using payment intent ID as session ID
		UserID:      "unknown",        // Would need to be extracted from metadata
		FeatureCode: "premium_feature",
		PlanID:      uuid.New(),
		Amount:      float64(paymentIntent.Amount) / 100.0, // Convert cents to dollars
		Currency:    string(paymentIntent.Currency),
		Status:      "completed",
		ExpiresAt:   nil,
		Metadata: map[string]interface{}{
			"payment_intent_id": paymentIntent.ID,
			"status":            paymentIntent.Status,
		},
	}

	a.logger.Info("Processed payment succeeded",
		zap.String("payment_intent_id", paymentIntent.ID),
		zap.Int64("amount", paymentIntent.Amount),
		zap.String("currency", string(paymentIntent.Currency)))

	return result, nil
}

// handlePaymentFailed handles payment_intent.payment_failed events
func (a *Adapter) handlePaymentFailed(event stripe.Event) (*billing.WebhookResult, error) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}

	result := &billing.WebhookResult{
		EventType:   string(billing.WebhookEventTypePaymentFailed),
		SessionID:   paymentIntent.ID,
		UserID:      "unknown",
		FeatureCode: "premium_feature",
		PlanID:      uuid.New(),
		Amount:      float64(paymentIntent.Amount) / 100.0, // Convert cents to dollars
		Currency:    string(paymentIntent.Currency),
		Status:      "failed",
		ExpiresAt:   nil,
		Metadata: map[string]interface{}{
			"payment_intent_id": paymentIntent.ID,
			"status":            paymentIntent.Status,
			"failure_reason":    paymentIntent.LastPaymentError,
		},
	}

	a.logger.Info("Processed payment failed",
		zap.String("payment_intent_id", paymentIntent.ID),
		zap.String("failure_reason", fmt.Sprintf("%v", paymentIntent.LastPaymentError)))

	return result, nil
}

// getFeatureCodeForPlan returns the appropriate feature code for a plan
func (a *Adapter) getFeatureCodeForPlan(planID string) string {
	switch planID {
	case "basic_monthly":
		return "basic_storage"
	case "pro_monthly":
		return "pro_storage"
	case "family_monthly":
		return "family_storage"
	// Legacy UUID support
	case "550e8400-e29b-41d4-a716-446655440001": // Basic Plan UUID
		return "basic_storage"
	case "550e8400-e29b-41d4-a716-446655440002": // Pro Plan UUID
		return "pro_storage"
	case "550e8400-e29b-41d4-a716-446655440003": // Family Plan UUID
		return "family_storage"
	default:
		return "premium_feature"
	}
}

// Close closes the Stripe adapter
func (a *Adapter) Close() error {
	// TODO: Implement cleanup if needed
	// Stripe doesn't require explicit connection cleanup

	a.logger.Info("Stripe adapter closed")
	return nil
}

// Helper function to safely get string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
	"go.uber.org/zap"
)

// PaymentProvider defines the interface for payment providers
type PaymentProvider interface {
	CreateCheckoutSession(ctx context.Context, params CheckoutSessionParams) (*CheckoutSession, error)
	ValidateWebhookSignature(payload []byte, signature string) error
	ParseWebhookEvent(payload []byte) (*WebhookEvent, error)
	ExtractPaymentIntentData(event *WebhookEvent) (*PaymentIntentData, error)
}

// StripeProvider implements payment provider using Stripe
type StripeProvider struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
	successURL     string
	cancelURL      string
	logger         *zap.Logger
}

// NewStripeProvider creates a new Stripe provider
func NewStripeProvider(secretKey, publishableKey, webhookSecret, successURL, cancelURL string, logger *zap.Logger) *StripeProvider {
	// Validate required parameters
	if secretKey == "" {
		panic("Stripe secret key is required")
	}
	if webhookSecret == "" {
		panic("Stripe webhook secret is required")
	}
	if successURL == "" {
		panic("Success URL is required")
	}
	if cancelURL == "" {
		panic("Cancel URL is required")
	}

	// Set the Stripe API key
	stripe.Key = secretKey

	return &StripeProvider{
		secretKey:      secretKey,
		publishableKey: publishableKey,
		webhookSecret:  webhookSecret,
		successURL:     successURL,
		cancelURL:      cancelURL,
		logger:         logger,
	}
}

// CheckoutSession represents a checkout session
type CheckoutSession struct {
	SessionID string `json:"session_id"`
	URL       string `json:"url"`
	Provider  string `json:"provider"`
}

// CreateCheckoutSession creates a Stripe checkout session
func (s *StripeProvider) CreateCheckoutSession(ctx context.Context, params CheckoutSessionParams) (*CheckoutSession, error) {
	// Validate input parameters
	if err := s.validateCheckoutParams(params); err != nil {
		s.logger.Error("Invalid checkout parameters",
			zap.Error(err),
			zap.String("user_id", params.UserID),
			zap.String("plan_id", params.PlanID))
		return nil, fmt.Errorf("invalid checkout parameters: %w", err)
	}

	// Generate order ID if not provided
	if params.OrderID == "" {
		params.OrderID = uuid.New().String()
	}

	s.logger.Info("Creating Stripe checkout session",
		zap.String("user_id", params.UserID),
		zap.String("plan_id", params.PlanID),
		zap.String("order_id", params.OrderID),
		zap.Int64("amount", params.Amount),
		zap.String("currency", params.Currency))

	// Create Stripe checkout session
	sessionParams := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(params.Currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(params.ProductName),
						Description: stripe.String(params.ProductDescription),
					},
					UnitAmount: stripe.Int64(params.Amount), // Amount in cents
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.successURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.cancelURL),
		Metadata: map[string]string{
			"user_id":  params.UserID,
			"plan_id":  params.PlanID,
			"order_id": params.OrderID,
		},
		// Add security headers
		BillingAddressCollection: stripe.String("auto"),
		CustomerCreation:         stripe.String("if_required"),
		// Enable automatic tax calculation if available
		AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{
			Enabled: stripe.Bool(true),
		},
	}

	// Set context timeout for Stripe API call
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create the session with context
	checkoutSession, err := session.New(sessionParams)
	if err != nil {
		s.logger.Error("Failed to create Stripe checkout session",
			zap.Error(err),
			zap.String("user_id", params.UserID),
			zap.String("plan_id", params.PlanID))
		return nil, fmt.Errorf("failed to create Stripe checkout session: %w", err)
	}

	s.logger.Info("Stripe checkout session created successfully",
		zap.String("session_id", checkoutSession.ID),
		zap.String("user_id", params.UserID),
		zap.String("plan_id", params.PlanID))

	return &CheckoutSession{
		SessionID: checkoutSession.ID,
		URL:       checkoutSession.URL,
		Provider:  "stripe",
	}, nil
}

// validateCheckoutParams validates checkout session parameters
func (s *StripeProvider) validateCheckoutParams(params CheckoutSessionParams) error {
	if params.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if params.PlanID == "" {
		return fmt.Errorf("plan_id is required")
	}
	if params.OrderID == "" {
		return fmt.Errorf("order_id is required")
	}
	if params.ProductName == "" {
		return fmt.Errorf("product_name is required")
	}
	if params.ProductDescription == "" {
		return fmt.Errorf("product_description is required")
	}
	if params.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if params.Currency == "" {
		return fmt.Errorf("currency is required")
	}
	// Validate currency code format (3 characters)
	if len(params.Currency) != 3 {
		return fmt.Errorf("currency must be a 3-character code")
	}

	// Validate amount is in reasonable range (not too small or too large)
	if params.Amount < 50 { // Minimum $0.50
		return fmt.Errorf("amount too small, minimum is $0.50")
	}
	if params.Amount > 99999999 { // Maximum $999,999.99
		return fmt.Errorf("amount too large, maximum is $999,999.99")
	}

	return nil
}

// CheckoutSessionParams contains parameters for creating a checkout session
type CheckoutSessionParams struct {
	UserID             string
	PlanID             string
	OrderID            string
	ProductName        string
	ProductDescription string
	Amount             int64
	Currency           string
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Created   int64           `json:"created"`
	Processed bool            `json:"processed"`
}

// ValidateWebhookSignature validates a Stripe webhook signature
func (s *StripeProvider) ValidateWebhookSignature(payload []byte, signature string) error {
	if len(payload) == 0 {
		s.logger.Error("Empty webhook payload received")
		return fmt.Errorf("empty payload")
	}
	if signature == "" {
		s.logger.Error("Empty webhook signature received")
		return fmt.Errorf("empty signature")
	}
	if s.webhookSecret == "" {
		s.logger.Error("Webhook secret not configured")
		return fmt.Errorf("webhook secret not configured")
	}

	// Parse the webhook signature
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		s.logger.Error("Failed to validate webhook signature",
			zap.Error(err),
			zap.String("signature", signature))
		return fmt.Errorf("failed to validate webhook signature: %w", err)
	}

	// Check if event is too old (replay attack prevention)
	eventAge := time.Now().Unix() - event.Created
	if eventAge > 300 { // 5 minutes
		s.logger.Warn("Webhook event is too old",
			zap.String("event_id", event.ID),
			zap.Int64("event_age_seconds", eventAge))
		return fmt.Errorf("webhook event is too old: %d seconds", eventAge)
	}

	// Additional validation: Check if event was created in the future
	if event.Created > time.Now().Unix()+60 { // Allow 1 minute clock skew
		s.logger.Warn("Webhook event created in the future",
			zap.String("event_id", event.ID),
			zap.Int64("created_timestamp", event.Created))
		return fmt.Errorf("webhook event created in the future: %d", event.Created)
	}

	// Validate event ID format (basic check)
	if event.ID == "" {
		s.logger.Error("Webhook event ID is empty")
		return fmt.Errorf("webhook event ID is empty")
	}

	s.logger.Debug("Webhook signature validated successfully",
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)))

	return nil
}

// ParseWebhookEvent parses a Stripe webhook event
func (s *StripeProvider) ParseWebhookEvent(payload []byte) (*WebhookEvent, error) {
	if len(payload) == 0 {
		s.logger.Error("Empty payload provided for webhook event parsing")
		return nil, fmt.Errorf("empty payload")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		s.logger.Error("Failed to parse webhook event JSON",
			zap.Error(err),
			zap.String("payload_preview", string(payload[:min(100, len(payload))])))
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}

	// Validate required fields
	if event.ID == "" {
		s.logger.Error("Webhook event missing ID field")
		return nil, fmt.Errorf("webhook event missing ID field")
	}
	if event.Type == "" {
		s.logger.Error("Webhook event missing type field",
			zap.String("event_id", event.ID))
		return nil, fmt.Errorf("webhook event missing type field")
	}

	s.logger.Debug("Webhook event parsed successfully",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type),
		zap.Int64("created", event.Created))

	return &event, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ExtractPaymentIntentData extracts payment intent data from a webhook event
func (s *StripeProvider) ExtractPaymentIntentData(event *WebhookEvent) (*PaymentIntentData, error) {
	if event == nil {
		s.logger.Error("Nil webhook event provided for payment intent data extraction")
		return nil, fmt.Errorf("webhook event is nil")
	}

	if len(event.Data) == 0 {
		s.logger.Error("Empty event data in webhook event",
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
		return nil, fmt.Errorf("empty event data")
	}

	// Parse the event data
	var eventData struct {
		Object struct {
			ID              string            `json:"id"`
			Amount          int64             `json:"amount"`
			Currency        string            `json:"currency"`
			Status          string            `json:"status"`
			Customer        string            `json:"customer"`
			Metadata        map[string]string `json:"metadata"`
			PaymentIntentID string            `json:"payment_intent"`
		} `json:"object"`
	}

	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		s.logger.Error("Failed to parse webhook event data",
			zap.Error(err),
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
		return nil, fmt.Errorf("failed to parse event data: %w", err)
	}

	// Validate required fields
	if eventData.Object.ID == "" {
		s.logger.Error("Missing object ID in webhook event data",
			zap.String("event_id", event.ID))
		return nil, fmt.Errorf("missing object ID in event data")
	}

	if eventData.Object.Amount <= 0 {
		s.logger.Error("Invalid amount in webhook event data",
			zap.String("event_id", event.ID),
			zap.Int64("amount", eventData.Object.Amount))
		return nil, fmt.Errorf("invalid amount: %d", eventData.Object.Amount)
	}

	if eventData.Object.Currency == "" {
		s.logger.Error("Missing currency in webhook event data",
			zap.String("event_id", event.ID))
		return nil, fmt.Errorf("missing currency in event data")
	}

	// Extract metadata with validation
	userID := eventData.Object.Metadata["user_id"]
	if userID == "" {
		s.logger.Error("Missing user_id in webhook event metadata",
			zap.String("event_id", event.ID))
		return nil, fmt.Errorf("missing user_id in metadata")
	}

	planID := eventData.Object.Metadata["plan_id"]
	if planID == "" {
		s.logger.Error("Missing plan_id in webhook event metadata",
			zap.String("event_id", event.ID),
			zap.String("user_id", userID))
		return nil, fmt.Errorf("missing plan_id in metadata")
	}

	orderID := eventData.Object.Metadata["order_id"]
	if orderID == "" {
		s.logger.Warn("Missing order_id in webhook event metadata, generating new one",
			zap.String("event_id", event.ID),
			zap.String("user_id", userID))
		orderID = uuid.New().String()
	}

	s.logger.Debug("Payment intent data extracted successfully",
		zap.String("event_id", event.ID),
		zap.String("payment_intent_id", eventData.Object.PaymentIntentID),
		zap.String("session_id", eventData.Object.ID),
		zap.String("user_id", userID),
		zap.String("plan_id", planID))

	return &PaymentIntentData{
		PaymentIntentID: eventData.Object.PaymentIntentID,
		SessionID:       eventData.Object.ID,
		Amount:          eventData.Object.Amount,
		Currency:        eventData.Object.Currency,
		Status:          eventData.Object.Status,
		CustomerID:      eventData.Object.Customer,
		UserID:          userID,
		PlanID:          planID,
		OrderID:         orderID,
	}, nil
}

// PaymentIntentData contains payment intent information
type PaymentIntentData struct {
	PaymentIntentID string
	SessionID       string
	Amount          int64
	Currency        string
	Status          string
	CustomerID      string
	UserID          string
	PlanID          string
	OrderID         string
}

// GetPublishableKey returns the publishable key
func (s *StripeProvider) GetPublishableKey() string {
	return s.publishableKey
}

package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StripeEvent represents a Stripe webhook event
type StripeEvent struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"`
	Created         int64           `json:"created"`
	Data            StripeEventData `json:"data"`
	Livemode        bool            `json:"livemode"`
	PendingWebhooks int             `json:"pending_webhooks"`
	Request         *StripeRequest  `json:"request"`
}

// StripeEventData contains the event data
type StripeEventData struct {
	Object map[string]interface{} `json:"object"`
}

// StripeRequest contains request information
type StripeRequest struct {
	ID             string `json:"id"`
	IdempotencyKey string `json:"idempotency_key"`
}

// WebhookResult represents the processed webhook data
type WebhookResult struct {
	UserID         string                 `json:"user_id"`
	FamilyID       *string                `json:"family_id,omitempty"`
	FeatureCode    string                 `json:"feature_code"`
	PlanID         uuid.UUID              `json:"plan_id"`
	PlanIDString   string                 `json:"plan_id_string"`
	SubscriptionID string                 `json:"subscription_id"`
	Amount         int64                  `json:"amount"`
	Currency       string                 `json:"currency"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	Status         string                 `json:"status"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// Parser handles webhook payload parsing
type Parser struct {
	supportedEvents map[string]bool
}

// NewParser creates a new webhook parser
func NewParser() *Parser {
	return &Parser{
		supportedEvents: map[string]bool{
			"payment_intent.succeeded":      true,
			"payment_intent.payment_failed": true,
			"invoice.payment_succeeded":     true,
			"invoice.payment_failed":        true,
			"customer.subscription.created": true,
			"customer.subscription.updated": true,
			"customer.subscription.deleted": true,
		},
	}
}

// ParseStripeWebhook parses a Stripe webhook payload
func (p *Parser) ParseStripeWebhook(payload []byte) (*WebhookResult, error) {
	var event StripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Check if event type is supported
	if !p.supportedEvents[event.Type] {
		return nil, fmt.Errorf("unsupported event type: %s", event.Type)
	}

	// Parse based on event type
	switch event.Type {
	case "payment_intent.succeeded":
		return p.parsePaymentIntentSucceeded(event)
	case "invoice.payment_succeeded":
		return p.parseInvoicePaymentSucceeded(event)
	case "customer.subscription.created":
		return p.parseSubscriptionCreated(event)
	case "customer.subscription.updated":
		return p.parseSubscriptionUpdated(event)
	case "customer.subscription.deleted":
		return p.parseSubscriptionDeleted(event)
	default:
		return nil, fmt.Errorf("event type not implemented: %s", event.Type)
	}
}

// parsePaymentIntentSucceeded parses payment_intent.succeeded events
func (p *Parser) parsePaymentIntentSucceeded(event StripeEvent) (*WebhookResult, error) {
	paymentIntent, ok := event.Data.Object["payment_intent"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payment_intent object")
	}

	// Extract metadata
	metadata, _ := paymentIntent["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Extract required fields
	userID, ok := metadata["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id in metadata")
	}

	// Feature code is no longer required in metadata - it will be determined from the plan
	featureCode := "" // Will be determined from plan

	planIDString, ok := metadata["plan_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing plan_id in metadata")
	}

	// Extract optional fields
	var familyID *string
	if fid, ok := metadata["family_id"].(string); ok && fid != "" {
		familyID = &fid
	}

	// Extract amount and currency
	amount, _ := paymentIntent["amount"].(float64)
	currency, _ := paymentIntent["currency"].(string)

	// Generate UUID from plan ID string
	planID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(planIDString))

	return &WebhookResult{
		UserID:         userID,
		FamilyID:       familyID,
		FeatureCode:    featureCode,
		PlanID:         planID,
		PlanIDString:   planIDString,
		SubscriptionID: "", // Payment intents don't have subscription IDs
		Amount:         int64(amount),
		Currency:       currency,
		ExpiresAt:      nil, // One-time payments don't expire
		Status:         "succeeded",
		Metadata:       metadata,
	}, nil
}

// parseInvoicePaymentSucceeded parses invoice.payment_succeeded events
func (p *Parser) parseInvoicePaymentSucceeded(event StripeEvent) (*WebhookResult, error) {
	invoice, ok := event.Data.Object["invoice"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid invoice object")
	}

	// Extract subscription ID
	subscriptionID, _ := invoice["subscription"].(string)
	if subscriptionID == "" {
		return nil, fmt.Errorf("missing subscription ID in invoice")
	}

	// Extract customer ID
	customerID, _ := invoice["customer"].(string)
	if customerID == "" {
		return nil, fmt.Errorf("missing customer ID in invoice")
	}

	// Extract amount and currency
	amount, _ := invoice["amount_paid"].(float64)
	currency, _ := invoice["currency"].(string)

	// Extract metadata
	metadata, _ := invoice["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Extract required fields from metadata
	userID, ok := metadata["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id in metadata")
	}

	// Feature code is no longer required in metadata - it will be determined from the plan
	featureCode := "" // Will be determined from plan

	planIDString, ok := metadata["plan_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing plan_id in metadata")
	}

	// Extract optional fields
	var familyID *string
	if fid, ok := metadata["family_id"].(string); ok && fid != "" {
		familyID = &fid
	}

	// Calculate expiration date (typically 1 month or 1 year from now)
	var expiresAt *time.Time
	if billingCycle, ok := metadata["billing_cycle"].(string); ok {
		now := time.Now()
		switch billingCycle {
		case "monthly":
			exp := now.AddDate(0, 1, 0)
			expiresAt = &exp
		case "yearly":
			exp := now.AddDate(1, 0, 0)
			expiresAt = &exp
		}
	}

	// Generate UUID from plan ID string
	planID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(planIDString))

	return &WebhookResult{
		UserID:         userID,
		FamilyID:       familyID,
		FeatureCode:    featureCode,
		PlanID:         planID,
		PlanIDString:   planIDString,
		SubscriptionID: subscriptionID,
		Amount:         int64(amount),
		Currency:       currency,
		ExpiresAt:      expiresAt,
		Status:         "succeeded",
		Metadata:       metadata,
	}, nil
}

// parseSubscriptionCreated parses customer.subscription.created events
func (p *Parser) parseSubscriptionCreated(event StripeEvent) (*WebhookResult, error) {
	subscription, ok := event.Data.Object["subscription"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid subscription object")
	}

	// Extract subscription ID
	subscriptionID, _ := subscription["id"].(string)
	if subscriptionID == "" {
		return nil, fmt.Errorf("missing subscription ID")
	}

	// Extract customer ID
	customerID, _ := subscription["customer"].(string)
	if customerID == "" {
		return nil, fmt.Errorf("missing customer ID")
	}

	// Extract metadata
	metadata, _ := subscription["metadata"].(map[string]interface{})
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Extract required fields from metadata
	userID, ok := metadata["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id in metadata")
	}

	// Feature code is no longer required in metadata - it will be determined from the plan
	featureCode := "" // Will be determined from plan

	planIDString, ok := metadata["plan_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing plan_id in metadata")
	}

	// Extract optional fields
	var familyID *string
	if fid, ok := metadata["family_id"].(string); ok && fid != "" {
		familyID = &fid
	}

	// Calculate expiration date
	var expiresAt *time.Time
	if currentPeriodEnd, ok := subscription["current_period_end"].(float64); ok {
		exp := time.Unix(int64(currentPeriodEnd), 0)
		expiresAt = &exp
	}

	// Generate UUID from plan ID string
	planID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(planIDString))

	return &WebhookResult{
		UserID:         userID,
		FamilyID:       familyID,
		FeatureCode:    featureCode,
		PlanID:         planID,
		PlanIDString:   planIDString,
		SubscriptionID: subscriptionID,
		Amount:         0,     // Subscriptions don't have immediate amounts
		Currency:       "usd", // Default currency
		ExpiresAt:      expiresAt,
		Status:         "active",
		Metadata:       metadata,
	}, nil
}

// parseSubscriptionUpdated parses customer.subscription.updated events
func (p *Parser) parseSubscriptionUpdated(event StripeEvent) (*WebhookResult, error) {
	// Similar to created but with updated status
	result, err := p.parseSubscriptionCreated(event)
	if err != nil {
		return nil, err
	}

	// Update status based on subscription state
	subscription, ok := event.Data.Object["subscription"].(map[string]interface{})
	if ok {
		if status, ok := subscription["status"].(string); ok {
			result.Status = status
		}
	}

	return result, nil
}

// parseSubscriptionDeleted parses customer.subscription.deleted events
func (p *Parser) parseSubscriptionDeleted(event StripeEvent) (*WebhookResult, error) {
	result, err := p.parseSubscriptionCreated(event)
	if err != nil {
		return nil, err
	}

	result.Status = "cancelled"
	return result, nil
}

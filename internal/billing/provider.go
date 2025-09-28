package billing

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Provider defines the interface for billing providers
type Provider interface {
	// CreateCheckoutSession creates a checkout session for a plan
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutSessionRequest) (*CreateCheckoutSessionResponse, error)

	// GetSession retrieves a checkout session by ID
	GetSession(ctx context.Context, sessionID string) (*Session, error)

	// CancelSession cancels a checkout session
	CancelSession(ctx context.Context, sessionID string) error

	// ValidateWebhook validates a webhook signature and payload
	ValidateWebhook(ctx context.Context, payload []byte, signature string) error

	// ParseWebhook parses webhook payload into a WebhookResult
	ParseWebhook(ctx context.Context, payload []byte) (*WebhookResult, error)

	// Close closes the provider connection
	Close() error
}

// CreateCheckoutSessionRequest represents a request to create a checkout session
type CreateCheckoutSessionRequest struct {
	PlanID      uuid.UUID         `json:"plan_id"`
	UserID      string            `json:"user_id"`
	FamilyID    *string           `json:"family_id,omitempty"`
	SuccessURL  string            `json:"success_url"`
	CancelURL   string            `json:"cancel_url"`
	CountryCode string            `json:"country_code,omitempty"` // ISO country code for pricing
	BasePrice   float64           `json:"base_price"`             // Base price in dollars
	Currency    string            `json:"currency"`               // Currency code
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CreateCheckoutSessionResponse represents the response from creating a checkout session
type CreateCheckoutSessionResponse struct {
	SessionID string    `json:"session_id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Session represents a checkout session
type Session struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	URL       string                 `json:"url,omitempty"`
	ExpiresAt time.Time              `json:"expires_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// WebhookResult represents the result of a billing webhook
type WebhookResult struct {
	EventType      string                 `json:"event_type"`
	SessionID      string                 `json:"session_id"`
	SubscriptionID string                 `json:"subscription_id"`
	UserID         string                 `json:"user_id"`
	FamilyID       *string                `json:"family_id,omitempty"`
	FeatureCode    string                 `json:"feature_code"`
	PlanID         uuid.UUID              `json:"plan_id"`
	PlanIDString   string                 `json:"plan_id_string"` // Original plan ID string for database
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	Status         string                 `json:"status"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SessionStatus represents the status of a checkout session
type SessionStatus string

const (
	SessionStatusOpen      SessionStatus = "open"
	SessionStatusComplete  SessionStatus = "complete"
	SessionStatusExpired   SessionStatus = "expired"
	SessionStatusCancelled SessionStatus = "cancelled"
)

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	WebhookEventTypeCheckoutSessionCompleted WebhookEventType = "checkout.session.completed"
	WebhookEventTypePaymentSucceeded         WebhookEventType = "payment.succeeded"
	WebhookEventTypePaymentFailed            WebhookEventType = "payment.failed"
	WebhookEventTypeSubscriptionCreated      WebhookEventType = "subscription.created"
	WebhookEventTypeSubscriptionUpdated      WebhookEventType = "subscription.updated"
	WebhookEventTypeSubscriptionCancelled    WebhookEventType = "subscription.cancelled"
)

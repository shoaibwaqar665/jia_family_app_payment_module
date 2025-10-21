package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Payment represents a payment transaction
type Payment struct {
	ID                    uuid.UUID       `json:"id"`
	Amount                int64           `json:"amount"` // Amount in cents
	Currency              string          `json:"currency"`
	Status                string          `json:"status"`
	PaymentMethod         string          `json:"payment_method"`
	CustomerID            string          `json:"customer_id"`
	OrderID               string          `json:"order_id"`
	Description           string          `json:"description"`
	StripePaymentIntentID *string         `json:"stripe_payment_intent_id,omitempty"`
	StripeSessionID       *string         `json:"stripe_session_id,omitempty"`
	Metadata              json.RawMessage `json:"metadata,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

// PaymentMethod represents the method used for payment
type PaymentMethod string

const (
	PaymentMethodCreditCard    PaymentMethod = "credit_card"
	PaymentMethodDebitCard     PaymentMethod = "debit_card"
	PaymentMethodBankTransfer  PaymentMethod = "bank_transfer"
	PaymentMethodDigitalWallet PaymentMethod = "digital_wallet"
)

// PaymentRequest represents a request to create a payment
type PaymentRequest struct {
	Amount        int64  `json:"amount" validate:"required,gt=0"`
	Currency      string `json:"currency" validate:"required,len=3"`
	PaymentMethod string `json:"payment_method" validate:"required"`
	CustomerID    string `json:"customer_id" validate:"required"`
	OrderID       string `json:"order_id" validate:"required"`
	Description   string `json:"description"`
}

// PaymentResponse represents a payment response
type PaymentResponse struct {
	ID            uuid.UUID `json:"id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	PaymentMethod string    `json:"payment_method"`
	CustomerID    string    `json:"customer_id"`
	OrderID       string    `json:"order_id"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
}

// IsValidStatus checks if the payment status is valid
func (p Payment) IsValidStatus() bool {
	switch p.Status {
	case string(PaymentStatusPending),
		string(PaymentStatusCompleted),
		string(PaymentStatusFailed),
		string(PaymentStatusCancelled),
		string(PaymentStatusRefunded):
		return true
	default:
		return false
	}
}

// IsValidPaymentMethod checks if the payment method is valid
func (p Payment) IsValidPaymentMethod() bool {
	switch p.PaymentMethod {
	case string(PaymentMethodCreditCard),
		string(PaymentMethodDebitCard),
		string(PaymentMethodBankTransfer),
		string(PaymentMethodDigitalWallet):
		return true
	default:
		return false
	}
}

// Plan represents a subscription plan
type Plan struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	FeatureCodes []string        `json:"feature_codes"`
	BillingCycle string          `json:"billing_cycle"`
	PriceCents   int64           `json:"price_cents"`
	Currency     string          `json:"currency"`
	MaxUsers     int32           `json:"max_users"`
	UsageLimits  json.RawMessage `json:"usage_limits"`
	Metadata     json.RawMessage `json:"metadata"`
	Active       bool            `json:"active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// Entitlement represents a user's entitlement to a feature
type Entitlement struct {
	ID             uuid.UUID       `json:"id"`
	UserID         string          `json:"user_id"`
	FamilyID       *string         `json:"family_id,omitempty"`
	Status         string          `json:"status"`
	FeatureCode    string          `json:"feature_code"`
	PlanID         uuid.UUID       `json:"plan_id"`
	SubscriptionID *string         `json:"subscription_id,omitempty"`
	GrantedAt      time.Time       `json:"granted_at"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	UsageLimits    json.RawMessage `json:"usage_limits"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// WebhookEvent represents a webhook event for idempotency
type WebhookEvent struct {
	ID          uuid.UUID  `json:"id"`
	EventID     string     `json:"event_id"`
	EventType   string     `json:"event_type"`
	Payload     []byte     `json:"payload"`
	Signature   string     `json:"signature"`
	Processed   bool       `json:"processed"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// OutboxEvent represents an event in the transactional outbox
type OutboxEvent struct {
	ID           string     `json:"id"`
	EventType    string     `json:"event_type"`
	Payload      []byte     `json:"payload"`
	Status       string     `json:"status"`
	RetryCount   int        `json:"retry_count"`
	CreatedAt    time.Time  `json:"created_at"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
}

package events

import (
	"context"
)

// DunningEvent represents a dunning event (moved from usecase to avoid import cycle)
type DunningEvent struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	FamilyID       *string                `json:"family_id,omitempty"`
	PaymentID      string                 `json:"payment_id"`
	SubscriptionID *string                `json:"subscription_id,omitempty"`
	EventType      string                 `json:"event_type"`
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	FailureReason  string                 `json:"failure_reason"`
	RetryCount     int                    `json:"retry_count"`
	NextRetryAt    *int64                 `json:"next_retry_at,omitempty"`
	Status         string                 `json:"status"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      int64                  `json:"created_at"`
	UpdatedAt      int64                  `json:"updated_at"`
}

// DunningPublisher defines the interface for publishing dunning events
type DunningPublisher interface {
	// PublishDunningEvent publishes a dunning event
	PublishDunningEvent(ctx context.Context, event *DunningEvent) error

	// PublishRetryAttempt publishes a retry attempt event
	PublishRetryAttempt(ctx context.Context, event *DunningEvent) error

	// PublishRetryResult publishes a retry result event
	PublishRetryResult(ctx context.Context, event *DunningEvent, success bool) error

	// PublishSubscriptionSuspended publishes a subscription suspension event
	PublishSubscriptionSuspended(ctx context.Context, subscriptionID, reason string) error

	// PublishDunningEscalated publishes a dunning escalation event
	PublishDunningEscalated(ctx context.Context, event *DunningEvent) error
}

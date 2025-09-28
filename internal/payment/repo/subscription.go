package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/payment/domain"
)

// SubscriptionRepository defines the interface for subscription data operations
type SubscriptionRepository interface {
	// Create creates a new subscription
	Create(ctx context.Context, sub domain.Subscription) (*domain.Subscription, error)

	// GetByID retrieves a subscription by ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)

	// GetByExternalID retrieves a subscription by external subscription ID
	GetByExternalID(ctx context.Context, externalID string) (*domain.Subscription, error)

	// GetByUserID retrieves all subscriptions for a user
	GetByUserID(ctx context.Context, userID string) ([]*domain.Subscription, error)

	// GetByStatus retrieves subscriptions with a specific status
	GetByStatus(ctx context.Context, status string) ([]*domain.Subscription, error)

	// Update updates an existing subscription
	Update(ctx context.Context, sub domain.Subscription) (*domain.Subscription, error)

	// Delete deletes a subscription
	Delete(ctx context.Context, id uuid.UUID) error

	// GetExpiringSubscriptions retrieves subscriptions expiring before a given date
	GetExpiringSubscriptions(ctx context.Context, beforeDate time.Time) ([]*domain.Subscription, error)

	// GetActiveSubscriptions retrieves all active subscriptions
	GetActiveSubscriptions(ctx context.Context) ([]*domain.Subscription, error)

	// GetSubscriptionsByPlan retrieves subscriptions for a specific plan
	GetSubscriptionsByPlan(ctx context.Context, planID uuid.UUID) ([]*domain.Subscription, error)
}

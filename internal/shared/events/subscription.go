package events

import (
	"context"

	"github.com/jia-app/paymentservice/internal/payment/domain"
)

// SubscriptionPublisher defines the interface for publishing subscription events
type SubscriptionPublisher interface {
	// PublishSubscriptionCreated publishes a subscription created event
	PublishSubscriptionCreated(ctx context.Context, sub *domain.Subscription) error

	// PublishSubscriptionStatusChanged publishes a subscription status change event
	PublishSubscriptionStatusChanged(ctx context.Context, sub *domain.Subscription, oldStatus, reason string) error

	// PublishSubscriptionRenewed publishes a subscription renewal event
	PublishSubscriptionRenewed(ctx context.Context, sub *domain.Subscription) error

	// PublishSubscriptionCancelled publishes a subscription cancellation event
	PublishSubscriptionCancelled(ctx context.Context, sub *domain.Subscription, reason string) error
}

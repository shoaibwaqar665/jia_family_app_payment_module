package events

import (
	"context"

	"github.com/jia-app/paymentservice/internal/payment/domain"
)

// UsagePublisher defines the interface for publishing usage events
type UsagePublisher interface {
	// PublishUsageTracked publishes a usage tracked event
	PublishUsageTracked(ctx context.Context, usage *domain.Usage) error

	// PublishQuotaExceeded publishes a quota exceeded event
	PublishQuotaExceeded(ctx context.Context, userID, featureCode, resourceType string, currentUsage, quotaLimit int64) error

	// PublishUsageReset publishes a usage reset event
	PublishUsageReset(ctx context.Context, userID, featureCode, resourceType string) error
}

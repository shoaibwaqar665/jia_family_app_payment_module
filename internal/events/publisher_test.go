package events

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/domain"
)

func TestNoopPublisher(t *testing.T) {
	publisher := NoopPublisher{}
	ctx := context.Background()

	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      "user123",
		FeatureCode: "premium",
		Status:      "active",
	}

	// Should always return nil without any side effects
	err := publisher.PublishEntitlementUpdated(ctx, entitlement, "created")
	if err != nil {
		t.Errorf("Expected no error from NoopPublisher, got: %v", err)
	}
}

func TestKafkaPublisher(t *testing.T) {
	// Create a test logger
	logger := zap.NewNop()

	publisher := NewKafkaPublisher("entitlements", logger)
	ctx := context.Background()

	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      "user456",
		FeatureCode: "basic",
		Status:      "active",
		PlanID:      uuid.New(),
	}

	// Should log the event and return nil
	err := publisher.PublishEntitlementUpdated(ctx, entitlement, "updated")
	if err != nil {
		t.Errorf("Expected no error from KafkaPublisher, got: %v", err)
	}
}

func TestKafkaPublisherConstructor(t *testing.T) {
	logger := zap.NewNop()
	topic := "test-topic"

	publisher := NewKafkaPublisher(topic, logger)

	if publisher.topic != topic {
		t.Errorf("Expected topic %s, got %s", topic, publisher.topic)
	}

	if publisher.logger != logger {
		t.Errorf("Expected logger to be set")
	}
}

func TestEntitlementPublisherInterface(t *testing.T) {
	// Test that both implementations satisfy the interface
	var _ EntitlementPublisher = NoopPublisher{}
	var _ EntitlementPublisher = &KafkaPublisher{}
}

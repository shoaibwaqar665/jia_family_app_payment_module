package events

import (
	"go.uber.org/zap"
)

// ExampleUsage demonstrates how to easily swap between different EntitlementPublisher implementations
func ExampleUsage() {
	// Create a logger
	logger := zap.NewNop()

	// Option 1: Use NoopPublisher for testing/development
	var publisher1 EntitlementPublisher = NoopPublisher{}

	// Option 2: Use KafkaPublisher for production
	var publisher2 EntitlementPublisher = NewKafkaPublisher("entitlements", logger)

	// Option 3: Configuration-based selection
	var publisher3 EntitlementPublisher
	env := "prod" // This could come from config

	switch env {
	case "prod":
		publisher3 = NewKafkaPublisher("entitlements", logger)
	case "dev", "test":
		publisher3 = NoopPublisher{}
	default:
		publisher3 = NoopPublisher{}
	}

	// All publishers implement the same interface, so they can be used interchangeably
	_ = publisher1
	_ = publisher2
	_ = publisher3
}

// ExampleWithConfig shows how to create a publisher based on configuration
func ExampleWithConfig(topic string, logger *zap.Logger, enableKafka bool) EntitlementPublisher {
	if enableKafka {
		return NewKafkaPublisher(topic, logger)
	}
	return NoopPublisher{}
}

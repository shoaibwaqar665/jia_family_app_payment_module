package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/jia-app/paymentservice/internal/domain"
	"go.uber.org/zap"
)

// Event represents a domain event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Aggregate string                 `json:"aggregate"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
	Version   int                    `json:"version"`
}

// Publisher defines the interface for publishing events
type Publisher interface {
	// Publish publishes an event
	Publish(ctx context.Context, event *Event) error

	// PublishBatch publishes multiple events
	PublishBatch(ctx context.Context, events []*Event) error

	// Close closes the publisher
	Close() error
}

// EventPublisher represents a concrete event publisher
type EventPublisher struct {
	kafkaPublisher *KafkaPublisher
	logger         *zap.Logger
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(kafkaPublisher *KafkaPublisher, logger *zap.Logger) *EventPublisher {
	return &EventPublisher{
		kafkaPublisher: kafkaPublisher,
		logger:         logger,
	}
}

// Publish publishes an event
func (p *EventPublisher) Publish(ctx context.Context, event *Event) error {
	if p.kafkaPublisher != nil {
		return p.kafkaPublisher.Publish(ctx, event)
	}

	// Fallback to console logging if no Kafka publisher
	eventJSON, _ := json.Marshal(event)
	p.logger.Info("Publishing event",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type),
		zap.String("event_data", string(eventJSON)))
	return nil
}

// PublishBatch publishes multiple events
func (p *EventPublisher) PublishBatch(ctx context.Context, events []*Event) error {
	if p.kafkaPublisher != nil {
		return p.kafkaPublisher.PublishBatch(ctx, events)
	}

	// Fallback to individual publishing
	for _, event := range events {
		if err := p.Publish(ctx, event); err != nil {
			return fmt.Errorf("failed to publish event %s: %w", event.ID, err)
		}
	}
	return nil
}

// Close closes the publisher
func (p *EventPublisher) Close() error {
	if p.kafkaPublisher != nil {
		return p.kafkaPublisher.Close()
	}
	return nil
}

// NewEvent creates a new event
func NewEvent(eventType, aggregate string, data map[string]interface{}) *Event {
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Aggregate: aggregate,
		Data:      data,
		Timestamp: getCurrentTimestamp(),
		Version:   1,
	}
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// getCurrentTimestamp returns the current timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// EntitlementPublisher defines the interface for publishing entitlement events
type EntitlementPublisher interface {
	PublishEntitlementUpdated(ctx context.Context, e domain.Entitlement, action string) error
}

// NoopPublisher is a no-operation publisher for testing and development
type NoopPublisher struct{}

// PublishEntitlementUpdated implements EntitlementPublisher for NoopPublisher
func (NoopPublisher) PublishEntitlementUpdated(ctx context.Context, e domain.Entitlement, action string) error {
	return nil
}

// KafkaPublisher is a Kafka-based event publisher
type KafkaPublisher struct {
	topic    string
	logger   *zap.Logger
	producer sarama.SyncProducer
	brokers  []string
}

// NewKafkaPublisher creates a new Kafka publisher
func NewKafkaPublisher(topic string, brokers []string, logger *zap.Logger) (*KafkaPublisher, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 10 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &KafkaPublisher{
		topic:    topic,
		logger:   logger,
		producer: producer,
		brokers:  brokers,
	}, nil
}

// PublishEntitlementUpdated implements EntitlementPublisher for KafkaPublisher
func (p *KafkaPublisher) PublishEntitlementUpdated(ctx context.Context, e domain.Entitlement, action string) error {
	event := &Event{
		ID:        fmt.Sprintf("entitlement_%s_%d", e.ID.String(), time.Now().UnixNano()),
		Type:      "entitlement.updated",
		Aggregate: "entitlement",
		Data: map[string]interface{}{
			"entitlement_id": e.ID.String(),
			"user_id":        e.UserID,
			"plan_id":        e.PlanID.String(),
			"feature_code":   e.FeatureCode,
			"status":         e.Status,
			"expires_at":     e.ExpiresAt.Format(time.RFC3339),
			"action":         action,
		},
		Timestamp: time.Now().Unix(),
		Version:   1,
	}

	return p.Publish(ctx, event)
}

// Publish publishes an event to Kafka
func (p *KafkaPublisher) Publish(ctx context.Context, event *Event) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.Aggregate),
		Value: sarama.ByteEncoder(eventJSON),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("event-type"),
				Value: []byte(event.Type),
			},
			{
				Key:   []byte("event-id"),
				Value: []byte(event.ID),
			},
		},
	}

	partition, offset, err := p.producer.SendMessage(message)
	if err != nil {
		p.logger.Error("Failed to publish event to Kafka",
			zap.Error(err),
			zap.String("topic", p.topic),
			zap.String("event_id", event.ID),
			zap.String("event_type", event.Type))
		return fmt.Errorf("failed to send message to Kafka: %w", err)
	}

	p.logger.Info("Event published to Kafka",
		zap.String("topic", p.topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.String("event_id", event.ID),
		zap.String("event_type", event.Type))

	return nil
}

// PublishBatch publishes multiple events to Kafka
func (p *KafkaPublisher) PublishBatch(ctx context.Context, events []*Event) error {
	messages := make([]*sarama.ProducerMessage, len(events))

	for i, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event %d: %w", i, err)
		}

		messages[i] = &sarama.ProducerMessage{
			Topic: p.topic,
			Key:   sarama.StringEncoder(event.Aggregate),
			Value: sarama.ByteEncoder(eventJSON),
			Headers: []sarama.RecordHeader{
				{
					Key:   []byte("event-type"),
					Value: []byte(event.Type),
				},
				{
					Key:   []byte("event-id"),
					Value: []byte(event.ID),
				},
			},
		}
	}

	err := p.producer.SendMessages(messages)
	if err != nil {
		p.logger.Error("Failed to publish batch to Kafka",
			zap.Error(err),
			zap.String("topic", p.topic),
			zap.Int("event_count", len(events)))
		return fmt.Errorf("failed to send batch to Kafka: %w", err)
	}

	p.logger.Info("Batch published to Kafka",
		zap.String("topic", p.topic),
		zap.Int("event_count", len(events)))

	return nil
}

// Close closes the Kafka producer
func (p *KafkaPublisher) Close() error {
	if err := p.producer.Close(); err != nil {
		p.logger.Error("Failed to close Kafka producer", zap.Error(err))
		return fmt.Errorf("failed to close Kafka producer: %w", err)
	}
	p.logger.Info("Kafka producer closed")
	return nil
}

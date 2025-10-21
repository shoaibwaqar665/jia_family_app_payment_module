package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/events"
	"github.com/jia-app/paymentservice/internal/log"
	"github.com/jia-app/paymentservice/internal/repository"
	"go.uber.org/zap"
)

// Worker processes events from the outbox table
type Worker struct {
	outboxRepo repository.OutboxRepository
	publisher  events.Publisher
	logger     *zap.Logger
	interval   time.Duration
	batchSize  int
}

// Config holds worker configuration
type Config struct {
	Interval  time.Duration // Interval between processing cycles
	BatchSize int           // Number of events to process per cycle
}

// DefaultConfig returns a default worker configuration
func DefaultConfig() Config {
	return Config{
		Interval:  5 * time.Second,
		BatchSize: 10,
	}
}

// NewWorker creates a new outbox worker
func NewWorker(
	outboxRepo repository.OutboxRepository,
	publisher events.Publisher,
	logger *zap.Logger,
	config Config,
) *Worker {
	return &Worker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		logger:     logger,
		interval:   config.Interval,
		batchSize:  config.BatchSize,
	}
}

// Start starts the outbox worker
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Starting outbox worker",
		zap.Duration("interval", w.interval),
		zap.Int("batch_size", w.batchSize))

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start
	if err := w.processBatch(ctx); err != nil {
		w.logger.Error("Failed to process initial batch", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Outbox worker stopping due to context cancellation")
			return ctx.Err()
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.logger.Error("Failed to process outbox batch", zap.Error(err))
			}
		}
	}
}

// processBatch processes a batch of outbox events
func (w *Worker) processBatch(ctx context.Context) error {
	// Get unprocessed events from outbox
	events, err := w.outboxRepo.GetPending(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed events: %w", err)
	}

	if len(events) == 0 {
		return nil // No events to process
	}

	w.logger.Info("Processing outbox batch",
		zap.Int("count", len(events)))

	// Process each event
	for _, event := range events {
		if err := w.processEvent(ctx, event); err != nil {
			w.logger.Error("Failed to process outbox event",
				zap.Error(err),
				zap.String("event_id", event.ID),
				zap.String("event_type", event.EventType))
			// Continue processing other events even if one fails
			continue
		}

		// Mark event as published
		if err := w.outboxRepo.MarkPublished(ctx, event.ID); err != nil {
			w.logger.Error("Failed to mark event as published",
				zap.Error(err),
				zap.String("event_id", event.ID))
			// Don't fail the entire batch if marking fails
		}
	}

	return nil
}

// processEvent processes a single outbox event
func (w *Worker) processEvent(ctx context.Context, event domain.OutboxEvent) error {
	// Parse event payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal event payload: %w", err)
	}

	// Create event for publishing
	publishEvent := &events.Event{
		ID:        event.ID,
		Type:      event.EventType,
		Aggregate: "payment",
		Data:      payload,
		Timestamp: event.CreatedAt.Unix(),
		Version:   1,
	}

	// Publish event
	if err := w.publisher.Publish(ctx, publishEvent); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Info(ctx, "Successfully published outbox event",
		zap.String("event_id", event.ID),
		zap.String("event_type", event.EventType))

	return nil
}

// Stop stops the outbox worker
func (w *Worker) Stop(ctx context.Context) error {
	w.logger.Info("Stopping outbox worker")

	// Process any remaining events before stopping
	if err := w.processBatch(ctx); err != nil {
		w.logger.Error("Failed to process final batch", zap.Error(err))
	}

	return nil
}

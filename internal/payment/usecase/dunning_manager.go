package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// DunningManager handles dunning management for failed payments
type DunningManager struct {
	paymentRepo      repo.PaymentRepository
	subscriptionRepo repo.SubscriptionRepository
	eventPublisher   events.DunningPublisher
}

// NewDunningManager creates a new dunning manager
func NewDunningManager(
	paymentRepo repo.PaymentRepository,
	subscriptionRepo repo.SubscriptionRepository,
	eventPublisher events.DunningPublisher,
) *DunningManager {
	return &DunningManager{
		paymentRepo:      paymentRepo,
		subscriptionRepo: subscriptionRepo,
		eventPublisher:   eventPublisher,
	}
}

// DunningEvent represents a dunning event
type DunningEvent struct {
	ID             uuid.UUID              `json:"id"`
	UserID         string                 `json:"user_id"`
	FamilyID       *string                `json:"family_id,omitempty"`
	PaymentID      string                 `json:"payment_id"`
	SubscriptionID *string                `json:"subscription_id,omitempty"`
	EventType      DunningEventType       `json:"event_type"`
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	FailureReason  string                 `json:"failure_reason"`
	RetryCount     int                    `json:"retry_count"`
	NextRetryAt    *time.Time             `json:"next_retry_at,omitempty"`
	Status         DunningStatus          `json:"status"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// DunningEventType represents the type of dunning event
type DunningEventType string

const (
	DunningEventTypePaymentFailed         DunningEventType = "payment_failed"
	DunningEventTypeRetryScheduled        DunningEventType = "retry_scheduled"
	DunningEventTypeRetryAttempted        DunningEventType = "retry_attempted"
	DunningEventTypeRetrySucceeded        DunningEventType = "retry_succeeded"
	DunningEventTypeRetryFailed           DunningEventType = "retry_failed"
	DunningEventTypeSubscriptionSuspended DunningEventType = "subscription_suspended"
	DunningEventTypeSubscriptionCancelled DunningEventType = "subscription_cancelled"
	DunningEventTypeDunningEscalated      DunningEventType = "dunning_escalated"
)

// DunningStatus represents the status of a dunning event
type DunningStatus string

const (
	DunningStatusActive    DunningStatus = "active"
	DunningStatusResolved  DunningStatus = "resolved"
	DunningStatusCancelled DunningStatus = "cancelled"
	DunningStatusEscalated DunningStatus = "escalated"
)

// DunningConfig holds configuration for dunning management
type DunningConfig struct {
	MaxRetryAttempts int             `json:"max_retry_attempts"`
	RetryIntervals   []time.Duration `json:"retry_intervals"`
	EscalationDelay  time.Duration   `json:"escalation_delay"`
	GracePeriod      time.Duration   `json:"grace_period"`
}

// DefaultDunningConfig returns a default dunning configuration
func DefaultDunningConfig() DunningConfig {
	return DunningConfig{
		MaxRetryAttempts: 3,
		RetryIntervals: []time.Duration{
			1 * time.Hour,  // First retry after 1 hour
			24 * time.Hour, // Second retry after 1 day
			72 * time.Hour, // Third retry after 3 days
		},
		EscalationDelay: 7 * 24 * time.Hour, // Escalate after 7 days
		GracePeriod:     24 * time.Hour,     // Grace period of 1 day
	}
}

// ProcessPaymentFailure processes a failed payment and initiates dunning
func (dm *DunningManager) ProcessPaymentFailure(ctx context.Context, req ProcessPaymentFailureRequest) error {
	// Validate input
	if req.PaymentID == "" {
		return status.Error(codes.InvalidArgument, "payment_id is required")
	}
	if req.UserID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Get payment details
	payment, err := dm.paymentRepo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get payment: %v", err)
	}

	if payment == nil {
		return status.Errorf(codes.NotFound, "payment not found: %s", req.PaymentID)
	}

	// Create dunning event
	dunningEvent := DunningEvent{
		ID:             uuid.New(),
		UserID:         req.UserID,
		FamilyID:       req.FamilyID,
		PaymentID:      req.PaymentID,
		SubscriptionID: req.SubscriptionID,
		EventType:      DunningEventTypePaymentFailed,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		FailureReason:  req.FailureReason,
		RetryCount:     0,
		Status:         DunningStatusActive,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Schedule first retry
	config := DefaultDunningConfig()
	if len(config.RetryIntervals) > 0 {
		nextRetry := time.Now().Add(config.RetryIntervals[0])
		dunningEvent.NextRetryAt = &nextRetry
	}

	// Store dunning event
	if err := dm.storeDunningEvent(ctx, dunningEvent); err != nil {
		return status.Errorf(codes.Internal, "failed to store dunning event: %v", err)
	}

	// Update payment status
	payment.Status = "failed"
	metadata := make(map[string]interface{})
	if len(payment.Metadata) > 0 {
		json.Unmarshal(payment.Metadata, &metadata)
	}
	metadata["failure_reason"] = req.FailureReason
	metadata["dunning_event_id"] = dunningEvent.ID.String()
	metadataBytes, _ := json.Marshal(metadata)
	payment.Metadata = metadataBytes
	if err := dm.paymentRepo.Update(ctx, payment); err != nil {
		log.L(ctx).Error("Failed to update payment status", zap.Error(err))
	}

	// Publish dunning event
	if dm.eventPublisher != nil {
		event := &events.DunningEvent{
			ID:             dunningEvent.ID.String(),
			UserID:         dunningEvent.UserID,
			FamilyID:       dunningEvent.FamilyID,
			PaymentID:      dunningEvent.PaymentID,
			SubscriptionID: dunningEvent.SubscriptionID,
			EventType:      string(dunningEvent.EventType),
			Amount:         dunningEvent.Amount,
			Currency:       dunningEvent.Currency,
			FailureReason:  dunningEvent.FailureReason,
			RetryCount:     dunningEvent.RetryCount,
			Status:         string(dunningEvent.Status),
			Metadata:       dunningEvent.Metadata,
			CreatedAt:      dunningEvent.CreatedAt.Unix(),
			UpdatedAt:      dunningEvent.UpdatedAt.Unix(),
		}
		if dunningEvent.NextRetryAt != nil {
			nextRetry := dunningEvent.NextRetryAt.Unix()
			event.NextRetryAt = &nextRetry
		}
		if err := dm.eventPublisher.PublishDunningEvent(ctx, event); err != nil {
			log.L(ctx).Warn("Failed to publish dunning event", zap.Error(err))
		}
	}

	log.Info(ctx, "Payment failure processed",
		zap.String("payment_id", req.PaymentID),
		zap.String("user_id", req.UserID),
		zap.String("failure_reason", req.FailureReason),
		zap.String("dunning_event_id", dunningEvent.ID.String()))

	return nil
}

// ProcessRetryAttempt processes a retry attempt for a failed payment
func (dm *DunningManager) ProcessRetryAttempt(ctx context.Context, req ProcessRetryAttemptRequest) error {
	// Get dunning event
	dunningEvent, err := dm.getDunningEvent(ctx, req.DunningEventID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get dunning event: %v", err)
	}

	if dunningEvent == nil {
		return status.Errorf(codes.NotFound, "dunning event not found: %s", req.DunningEventID)
	}

	// Check if retry is allowed
	config := DefaultDunningConfig()
	if dunningEvent.RetryCount >= config.MaxRetryAttempts {
		return status.Errorf(codes.FailedPrecondition, "maximum retry attempts exceeded")
	}

	// Update retry count
	dunningEvent.RetryCount++
	dunningEvent.EventType = DunningEventTypeRetryAttempted
	dunningEvent.UpdatedAt = time.Now()

	// Store updated event
	if err := dm.storeDunningEvent(ctx, *dunningEvent); err != nil {
		return status.Errorf(codes.Internal, "failed to update dunning event: %v", err)
	}

	// Publish retry attempt event
	if dm.eventPublisher != nil {
		event := &events.DunningEvent{
			ID:             dunningEvent.ID.String(),
			UserID:         dunningEvent.UserID,
			FamilyID:       dunningEvent.FamilyID,
			PaymentID:      dunningEvent.PaymentID,
			SubscriptionID: dunningEvent.SubscriptionID,
			EventType:      string(dunningEvent.EventType),
			Amount:         dunningEvent.Amount,
			Currency:       dunningEvent.Currency,
			FailureReason:  dunningEvent.FailureReason,
			RetryCount:     dunningEvent.RetryCount,
			Status:         string(dunningEvent.Status),
			Metadata:       dunningEvent.Metadata,
			CreatedAt:      dunningEvent.CreatedAt.Unix(),
			UpdatedAt:      dunningEvent.UpdatedAt.Unix(),
		}
		if dunningEvent.NextRetryAt != nil {
			nextRetry := dunningEvent.NextRetryAt.Unix()
			event.NextRetryAt = &nextRetry
		}
		if err := dm.eventPublisher.PublishRetryAttempt(ctx, event); err != nil {
			log.L(ctx).Warn("Failed to publish retry attempt event", zap.Error(err))
		}
	}

	log.Info(ctx, "Retry attempt processed",
		zap.String("dunning_event_id", req.DunningEventID.String()),
		zap.Int("retry_count", dunningEvent.RetryCount))

	return nil
}

// ProcessRetryResult processes the result of a retry attempt
func (dm *DunningManager) ProcessRetryResult(ctx context.Context, req ProcessRetryResultRequest) error {
	// Get dunning event
	dunningEvent, err := dm.getDunningEvent(ctx, req.DunningEventID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get dunning event: %v", err)
	}

	if dunningEvent == nil {
		return status.Errorf(codes.NotFound, "dunning event not found: %s", req.DunningEventID)
	}

	config := DefaultDunningConfig()

	if req.Success {
		// Retry succeeded
		dunningEvent.EventType = DunningEventTypeRetrySucceeded
		dunningEvent.Status = DunningStatusResolved
		dunningEvent.NextRetryAt = nil

		// Update payment status
		payment, err := dm.paymentRepo.GetByID(ctx, dunningEvent.PaymentID)
		if err == nil && payment != nil {
			payment.Status = "succeeded"
			metadata := make(map[string]interface{})
			if len(payment.Metadata) > 0 {
				json.Unmarshal(payment.Metadata, &metadata)
			}
			metadata["retry_succeeded"] = true
			metadata["retry_count"] = dunningEvent.RetryCount
			metadataBytes, _ := json.Marshal(metadata)
			payment.Metadata = metadataBytes
			dm.paymentRepo.Update(ctx, payment)
		}

		log.Info(ctx, "Retry succeeded",
			zap.String("dunning_event_id", req.DunningEventID.String()),
			zap.String("payment_id", dunningEvent.PaymentID))

	} else {
		// Retry failed
		dunningEvent.EventType = DunningEventTypeRetryFailed
		dunningEvent.FailureReason = req.FailureReason

		// Check if we should schedule another retry
		if dunningEvent.RetryCount < config.MaxRetryAttempts {
			// Schedule next retry
			retryIndex := dunningEvent.RetryCount
			if retryIndex < len(config.RetryIntervals) {
				nextRetry := time.Now().Add(config.RetryIntervals[retryIndex])
				dunningEvent.NextRetryAt = &nextRetry
			}
		} else {
			// Max retries exceeded, escalate
			dunningEvent.Status = DunningStatusEscalated
			dunningEvent.NextRetryAt = nil
			dunningEvent.EventType = DunningEventTypeDunningEscalated

			// Suspend subscription if applicable
			if dunningEvent.SubscriptionID != nil {
				if err := dm.suspendSubscription(ctx, *dunningEvent.SubscriptionID, "payment_failure"); err != nil {
					log.L(ctx).Error("Failed to suspend subscription", zap.Error(err))
				}
			}
		}
	}

	dunningEvent.UpdatedAt = time.Now()

	// Store updated event
	if err := dm.storeDunningEvent(ctx, *dunningEvent); err != nil {
		return status.Errorf(codes.Internal, "failed to update dunning event: %v", err)
	}

	// Publish retry result event
	if dm.eventPublisher != nil {
		event := &events.DunningEvent{
			ID:             dunningEvent.ID.String(),
			UserID:         dunningEvent.UserID,
			FamilyID:       dunningEvent.FamilyID,
			PaymentID:      dunningEvent.PaymentID,
			SubscriptionID: dunningEvent.SubscriptionID,
			EventType:      string(dunningEvent.EventType),
			Amount:         dunningEvent.Amount,
			Currency:       dunningEvent.Currency,
			FailureReason:  dunningEvent.FailureReason,
			RetryCount:     dunningEvent.RetryCount,
			Status:         string(dunningEvent.Status),
			Metadata:       dunningEvent.Metadata,
			CreatedAt:      dunningEvent.CreatedAt.Unix(),
			UpdatedAt:      dunningEvent.UpdatedAt.Unix(),
		}
		if dunningEvent.NextRetryAt != nil {
			nextRetry := dunningEvent.NextRetryAt.Unix()
			event.NextRetryAt = &nextRetry
		}
		if err := dm.eventPublisher.PublishRetryResult(ctx, event, req.Success); err != nil {
			log.L(ctx).Warn("Failed to publish retry result event", zap.Error(err))
		}
	}

	return nil
}

// GetDunningEvents returns dunning events for a user
func (dm *DunningManager) GetDunningEvents(ctx context.Context, userID string, status DunningStatus) ([]DunningEvent, error) {
	// This would typically query the dunning events repository
	// For now, return empty slice
	return []DunningEvent{}, nil
}

// CancelDunningEvent cancels a dunning event
func (dm *DunningManager) CancelDunningEvent(ctx context.Context, dunningEventID uuid.UUID, reason string) error {
	// Get dunning event
	dunningEvent, err := dm.getDunningEvent(ctx, dunningEventID)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get dunning event: %v", err)
	}

	if dunningEvent == nil {
		return status.Errorf(codes.NotFound, "dunning event not found: %s", dunningEventID.String())
	}

	// Cancel the event
	dunningEvent.Status = DunningStatusCancelled
	dunningEvent.NextRetryAt = nil
	dunningEvent.Metadata["cancellation_reason"] = reason
	dunningEvent.UpdatedAt = time.Now()

	// Store updated event
	if err := dm.storeDunningEvent(ctx, *dunningEvent); err != nil {
		return status.Errorf(codes.Internal, "failed to update dunning event: %v", err)
	}

	log.Info(ctx, "Dunning event cancelled",
		zap.String("dunning_event_id", dunningEventID.String()),
		zap.String("reason", reason))

	return nil
}

// Helper methods

// storeDunningEvent stores a dunning event (placeholder implementation)
func (dm *DunningManager) storeDunningEvent(ctx context.Context, event DunningEvent) error {
	// This would typically store in a dunning events repository
	// For now, just log the event
	log.L(ctx).Info("Storing dunning event",
		zap.String("event_id", event.ID.String()),
		zap.String("event_type", string(event.EventType)),
		zap.String("status", string(event.Status)))
	return nil
}

// getDunningEvent retrieves a dunning event (placeholder implementation)
func (dm *DunningManager) getDunningEvent(ctx context.Context, eventID uuid.UUID) (*DunningEvent, error) {
	// This would typically query the dunning events repository
	// For now, return nil
	return nil, nil
}

// suspendSubscription suspends a subscription
func (dm *DunningManager) suspendSubscription(ctx context.Context, subscriptionID string, reason string) error {
	// This would typically use the subscription lifecycle manager
	// For now, just log the action
	log.L(ctx).Info("Suspending subscription",
		zap.String("subscription_id", subscriptionID),
		zap.String("reason", reason))
	return nil
}

// Request types

type ProcessPaymentFailureRequest struct {
	PaymentID      string                 `json:"payment_id"`
	UserID         string                 `json:"user_id"`
	FamilyID       *string                `json:"family_id,omitempty"`
	SubscriptionID *string                `json:"subscription_id,omitempty"`
	FailureReason  string                 `json:"failure_reason"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type ProcessRetryAttemptRequest struct {
	DunningEventID uuid.UUID `json:"dunning_event_id"`
}

type ProcessRetryResultRequest struct {
	DunningEventID uuid.UUID `json:"dunning_event_id"`
	Success        bool      `json:"success"`
	FailureReason  string    `json:"failure_reason,omitempty"`
}

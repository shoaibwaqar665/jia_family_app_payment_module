package subscription

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// LifecycleManager handles subscription lifecycle operations
type LifecycleManager struct {
	subscriptionRepo repo.SubscriptionRepository
	entitlementRepo  repo.EntitlementRepository
	eventPublisher   events.SubscriptionPublisher
}

// NewLifecycleManager creates a new subscription lifecycle manager
func NewLifecycleManager(
	subscriptionRepo repo.SubscriptionRepository,
	entitlementRepo repo.EntitlementRepository,
	eventPublisher events.SubscriptionPublisher,
) *LifecycleManager {
	return &LifecycleManager{
		subscriptionRepo: subscriptionRepo,
		entitlementRepo:  entitlementRepo,
		eventPublisher:   eventPublisher,
	}
}

// CreateSubscription creates a new subscription
func (lm *LifecycleManager) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*domain.Subscription, error) {
	subscription := domain.Subscription{
		ID:                     uuid.New(),
		UserID:                 req.UserID,
		FamilyID:               req.FamilyID,
		PlanID:                 req.PlanID,
		Status:                 domain.SubscriptionStatusActive,
		CurrentPeriodStart:     req.CurrentPeriodStart,
		CurrentPeriodEnd:       req.CurrentPeriodEnd,
		CancelAtPeriodEnd:      false,
		ExternalSubscriptionID: req.ExternalSubscriptionID,
		Metadata:               req.Metadata,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	savedSubscription, err := lm.subscriptionRepo.Create(ctx, subscription)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create subscription: %v", err)
	}

	// Publish subscription created event
	if lm.eventPublisher != nil {
		if err := lm.eventPublisher.PublishSubscriptionCreated(ctx, savedSubscription); err != nil {
			log.Warn(ctx, "Failed to publish subscription created event", zap.Error(err))
		}
	}

	log.Info(ctx, "Subscription created",
		zap.String("subscription_id", savedSubscription.ID.String()),
		zap.String("user_id", savedSubscription.UserID),
		zap.String("plan_id", savedSubscription.PlanID.String()))

	return savedSubscription, nil
}

// UpdateStatus updates subscription status with proper lifecycle transitions
func (lm *LifecycleManager) UpdateStatus(ctx context.Context, subscriptionID uuid.UUID, newStatus string, reason string) error {
	subscription, err := lm.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "subscription not found: %v", err)
	}

	// Validate status transition
	if !lm.isValidStatusTransition(subscription.Status, newStatus) {
		return status.Errorf(codes.InvalidArgument, "invalid status transition from %s to %s", subscription.Status, newStatus)
	}

	oldStatus := subscription.Status
	subscription.Status = newStatus
	subscription.UpdatedAt = time.Now()

	// Handle specific status transitions
	switch newStatus {
	case domain.SubscriptionStatusCancelled:
		subscription.CancelledAt = &subscription.UpdatedAt
	case domain.SubscriptionStatusExpired:
		// Expired subscriptions should have their entitlements revoked
		if err := lm.revokeEntitlements(ctx, subscription); err != nil {
			log.Error(ctx, "Failed to revoke entitlements for expired subscription", zap.Error(err))
		}
	}

	// Update subscription in database
	updatedSubscription, err := lm.subscriptionRepo.Update(ctx, *subscription)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to update subscription: %v", err)
	}

	// Publish status change event
	if lm.eventPublisher != nil {
		if err := lm.eventPublisher.PublishSubscriptionStatusChanged(ctx, updatedSubscription, oldStatus, reason); err != nil {
			log.Warn(ctx, "Failed to publish subscription status changed event", zap.Error(err))
		}
	}

	log.Info(ctx, "Subscription status updated",
		zap.String("subscription_id", subscriptionID.String()),
		zap.String("old_status", oldStatus),
		zap.String("new_status", newStatus),
		zap.String("reason", reason))

	return nil
}

// ProcessPaymentFailure handles failed payment scenarios
func (lm *LifecycleManager) ProcessPaymentFailure(ctx context.Context, subscriptionID uuid.UUID, failureReason string) error {
	subscription, err := lm.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "subscription not found: %v", err)
	}

	// Determine appropriate status based on failure reason and current status
	var newStatus string
	switch subscription.Status {
	case domain.SubscriptionStatusActive:
		// First failure - move to past_due
		newStatus = domain.SubscriptionStatusPastDue
	case domain.SubscriptionStatusPastDue:
		// Second failure - move to suspended
		newStatus = domain.SubscriptionStatusSuspended
	case domain.SubscriptionStatusSuspended:
		// Third failure - cancel subscription
		newStatus = domain.SubscriptionStatusCancelled
	default:
		// Already in terminal state
		return nil
	}

	return lm.UpdateStatus(ctx, subscriptionID, newStatus, failureReason)
}

// ProcessPaymentSuccess handles successful payment scenarios
func (lm *LifecycleManager) ProcessPaymentSuccess(ctx context.Context, subscriptionID uuid.UUID) error {
	subscription, err := lm.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "subscription not found: %v", err)
	}

	// Only reactivate if currently past_due or suspended
	if subscription.Status == domain.SubscriptionStatusPastDue || subscription.Status == domain.SubscriptionStatusSuspended {
		return lm.UpdateStatus(ctx, subscriptionID, domain.SubscriptionStatusActive, "payment_successful")
	}

	return nil
}

// CancelSubscription cancels a subscription
func (lm *LifecycleManager) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, reason string) error {
	return lm.UpdateStatus(ctx, subscriptionID, domain.SubscriptionStatusCancelled, reason)
}

// RenewSubscription renews a subscription for the next period
func (lm *LifecycleManager) RenewSubscription(ctx context.Context, subscriptionID uuid.UUID, newPeriodEnd time.Time) error {
	subscription, err := lm.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return status.Errorf(codes.NotFound, "subscription not found: %v", err)
	}

	// Only renew active subscriptions
	if subscription.Status != domain.SubscriptionStatusActive {
		return status.Errorf(codes.InvalidArgument, "cannot renew subscription with status %s", subscription.Status)
	}

	subscription.CurrentPeriodStart = subscription.CurrentPeriodEnd
	subscription.CurrentPeriodEnd = newPeriodEnd
	subscription.UpdatedAt = time.Now()

	updatedSubscription, err := lm.subscriptionRepo.Update(ctx, *subscription)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to renew subscription: %v", err)
	}

	// Publish renewal event
	if lm.eventPublisher != nil {
		if err := lm.eventPublisher.PublishSubscriptionRenewed(ctx, updatedSubscription); err != nil {
			log.Warn(ctx, "Failed to publish subscription renewed event", zap.Error(err))
		}
	}

	log.Info(ctx, "Subscription renewed",
		zap.String("subscription_id", subscriptionID.String()),
		zap.Time("new_period_end", newPeriodEnd))

	return nil
}

// GetExpiringSubscriptions returns subscriptions that are expiring soon
func (lm *LifecycleManager) GetExpiringSubscriptions(ctx context.Context, withinDays int) ([]*domain.Subscription, error) {
	cutoffDate := time.Now().AddDate(0, 0, withinDays)
	return lm.subscriptionRepo.GetExpiringSubscriptions(ctx, cutoffDate)
}

// GetSubscriptionsByStatus returns subscriptions with a specific status
func (lm *LifecycleManager) GetSubscriptionsByStatus(ctx context.Context, status string) ([]*domain.Subscription, error) {
	return lm.subscriptionRepo.GetByStatus(ctx, status)
}

// Helper methods

// isValidStatusTransition validates if a status transition is allowed
func (lm *LifecycleManager) isValidStatusTransition(from, to string) bool {
	validTransitions := map[string][]string{
		domain.SubscriptionStatusActive:    {domain.SubscriptionStatusPastDue, domain.SubscriptionStatusSuspended, domain.SubscriptionStatusCancelled},
		domain.SubscriptionStatusPastDue:   {domain.SubscriptionStatusActive, domain.SubscriptionStatusSuspended, domain.SubscriptionStatusCancelled},
		domain.SubscriptionStatusSuspended: {domain.SubscriptionStatusActive, domain.SubscriptionStatusCancelled},
		domain.SubscriptionStatusCancelled: {domain.SubscriptionStatusExpired},
		domain.SubscriptionStatusExpired:   {}, // Terminal state
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}

	return false
}

// revokeEntitlements revokes all entitlements for a subscription
func (lm *LifecycleManager) revokeEntitlements(ctx context.Context, subscription *domain.Subscription) error {
	// Get all entitlements for this subscription
	entitlements, err := lm.entitlementRepo.GetBySubscriptionID(ctx, subscription.ExternalSubscriptionID)
	if err != nil {
		return err
	}

	// Revoke each entitlement
	for _, entitlement := range entitlements {
		entitlement.Status = "revoked"
		entitlement.UpdatedAt = time.Now()

		if _, err := lm.entitlementRepo.Update(ctx, entitlement); err != nil {
			log.Error(ctx, "Failed to revoke entitlement",
				zap.String("entitlement_id", entitlement.ID.String()),
				zap.Error(err))
		}
	}

	return nil
}

// Request types

type CreateSubscriptionRequest struct {
	UserID                 string                 `json:"user_id"`
	FamilyID               *string                `json:"family_id,omitempty"`
	PlanID                 uuid.UUID              `json:"plan_id"`
	CurrentPeriodStart     time.Time              `json:"current_period_start"`
	CurrentPeriodEnd       time.Time              `json:"current_period_end"`
	ExternalSubscriptionID string                 `json:"external_subscription_id"`
	Metadata               map[string]interface{} `json:"metadata"`
}

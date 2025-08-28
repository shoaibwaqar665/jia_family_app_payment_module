package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// EntitlementUseCase provides business logic for entitlement operations
type EntitlementUseCase struct {
	entitlementRepo      repo.EntitlementRepository
	cache                *cache.Cache // Can be nil if Redis is not available
	entitlementPublisher events.EntitlementPublisher
}

// NewEntitlementUseCase creates a new entitlement use case
func NewEntitlementUseCase(
	entitlementRepo repo.EntitlementRepository,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
) *EntitlementUseCase {
	return &EntitlementUseCase{
		entitlementRepo:      entitlementRepo,
		cache:                cache,
		entitlementPublisher: entitlementPublisher,
	}
}

// CheckEntitlement checks if a user has access to a specific feature
func (uc *EntitlementUseCase) CheckEntitlement(ctx context.Context, userID, featureCode string) (*CheckEntitlementResponse, error) {
	// Use authenticated user_id from context if userID is empty
	if userID == "" {
		if contextUserID := extractUserIDFromContext(ctx); contextUserID != "" {
			userID = contextUserID
		} else {
			return nil, status.Error(codes.InvalidArgument, "user_id is required")
		}
	}

	// Validate input
	if featureCode == "" {
		return nil, status.Error(codes.InvalidArgument, "feature_code is required")
	}

	// Try Redis cache first
	if uc.cache != nil {
		cachedEnt, found, err := uc.cache.GetEntitlement(ctx, userID, featureCode)
		if err != nil {
			log.Warn(ctx, "Failed to get entitlement from cache",
				zap.Error(err), zap.String("user_id", userID), zap.String("feature_code", featureCode))
		} else if found {
			// Check if it's a negative cache result
			if isNegative, err := uc.cache.IsEntitlementNotFound(ctx, userID, featureCode); err == nil && isNegative {
				return &CheckEntitlementResponse{
					Allowed:     false,
					Entitlement: nil,
				}, nil
			}

			// Validate cached entitlement is still active and not expired
			if isValidEntitlement(cachedEnt) {
				return &CheckEntitlementResponse{
					Allowed:     true,
					Entitlement: cachedEnt,
				}, nil
			}

			// Cached entitlement is invalid, evict from cache and fallback to repo
			uc.cache.DeleteEntitlement(ctx, userID, featureCode)
		}
	}

	// Fallback to repository
	entitlement, found, err := uc.entitlementRepo.Check(ctx, userID, featureCode)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check entitlement: %v", err)
	}

	if !found {
		// Cache negative result
		if uc.cache != nil {
			uc.cache.SetEntitlementNotFound(ctx, userID, featureCode)
		}
		return &CheckEntitlementResponse{
			Allowed:     false,
			Entitlement: nil,
		}, nil
	}

	// Validate entitlement is active and not expired
	if !isValidEntitlement(&entitlement) {
		// Cache negative result for invalid entitlements
		if uc.cache != nil {
			uc.cache.SetEntitlementNotFound(ctx, userID, featureCode)
		}
		return &CheckEntitlementResponse{
			Allowed:     false,
			Entitlement: nil,
		}, nil
	}

	// Cache valid entitlement
	if uc.cache != nil {
		uc.cache.SetEntitlement(ctx, entitlement, 0) // Use default TTL
	}

	return &CheckEntitlementResponse{
		Allowed:     true,
		Entitlement: &entitlement,
	}, nil
}

// ListUserEntitlements lists all entitlements for a user
func (uc *EntitlementUseCase) ListUserEntitlements(ctx context.Context, userID string) ([]*domain.Entitlement, error) {
	// Use authenticated user_id from context if userID is empty
	if userID == "" {
		if contextUserID := extractUserIDFromContext(ctx); contextUserID != "" {
			userID = contextUserID
		} else {
			return nil, status.Error(codes.InvalidArgument, "user_id is required")
		}
	}

	entitlements, err := uc.entitlementRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user entitlements: %v", err)
	}

	// Convert to pointers for consistency
	result := make([]*domain.Entitlement, len(entitlements))
	for i := range entitlements {
		result[i] = &entitlements[i]
	}

	return result, nil
}

// CreateEntitlement creates a new entitlement
func (uc *EntitlementUseCase) CreateEntitlement(ctx context.Context, userID, featureCode string, planID uuid.UUID, expiresAt *time.Time) (*domain.Entitlement, error) {
	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      userID,
		FeatureCode: featureCode,
		PlanID:      planID,
		Status:      "active",
		GrantedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	savedEntitlement, err := uc.entitlementRepo.Insert(ctx, entitlement)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create entitlement: %v", err)
	}

	// Publish entitlement.updated event
	if uc.entitlementPublisher != nil {
		if err := uc.entitlementPublisher.PublishEntitlementUpdated(ctx, savedEntitlement, "created"); err != nil {
			log.Error(ctx, "Failed to publish entitlement.updated event", zap.Error(err))
		}
	}

	// Evict cache
	if uc.cache != nil {
		uc.cache.DeleteEntitlement(ctx, savedEntitlement.UserID, savedEntitlement.FeatureCode)
	}

	return &savedEntitlement, nil
}

// Helper types for responses
type CheckEntitlementResponse struct {
	Allowed     bool                `json:"allowed"`
	Entitlement *domain.Entitlement `json:"entitlement,omitempty"`
}

// Helper functions
func extractUserIDFromContext(ctx context.Context) string {
	if userID := ctx.Value(log.UserIDKey); userID != nil {
		if uid, ok := userID.(string); ok {
			return uid
		}
	}
	return ""
}

func isValidEntitlement(ent *domain.Entitlement) bool {
	if ent == nil {
		return false
	}

	// Check if status is active
	if ent.Status != "active" {
		return false
	}

	// Check if not expired
	if ent.ExpiresAt != nil && ent.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

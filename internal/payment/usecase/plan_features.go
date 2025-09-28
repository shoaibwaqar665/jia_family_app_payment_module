package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/log"
	"go.uber.org/zap"
)

// PlanFeatureService handles plan-to-feature mapping and entitlement creation
type PlanFeatureService struct {
	planRepo repo.PlanRepository
	logger   *zap.Logger
}

// NewPlanFeatureService creates a new plan feature service
func NewPlanFeatureService(planRepo repo.PlanRepository) *PlanFeatureService {
	return &PlanFeatureService{
		planRepo: planRepo,
		logger:   log.L(context.Background()),
	}
}

// GrantEntitlementsForPlan grants all entitlements for a given plan
func (pfs *PlanFeatureService) GrantEntitlementsForPlan(ctx context.Context, userID string, planIDString string, familyID *string, subscriptionID *string, expiresAt *time.Time) ([]string, error) {
	// Get plan details from database
	plan, err := pfs.planRepo.GetByID(ctx, planIDString)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %s: %w", planIDString, err)
	}

	// Convert plan ID string to UUID
	planUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(planIDString))

	grantedFeatures := make([]string, 0, len(plan.FeatureCodes))

	// Grant entitlements for each feature in the plan
	for _, featureCode := range plan.FeatureCodes {
		// Create entitlement for this feature
		entitlement := domain.Entitlement{
			ID:          uuid.New(),
			UserID:      userID,
			FamilyID:    familyID,
			FeatureCode: featureCode,
			PlanID:      planUUID,
			Status:      "active",
			GrantedAt:   time.Now(),
			ExpiresAt:   expiresAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Set subscription ID if provided
		if subscriptionID != nil {
			entitlement.SubscriptionID = subscriptionID
		}

		// Insert entitlement (this will be handled by the entitlement use case)
		grantedFeatures = append(grantedFeatures, featureCode)

		pfs.logger.Info("Granting entitlement",
			zap.String("user_id", userID),
			zap.String("plan_id", planIDString),
			zap.String("feature_code", featureCode),
			zap.String("family_id", getStringValue(familyID)))
	}

	return grantedFeatures, nil
}

// GetPlanFeatures returns the feature codes for a given plan
func (pfs *PlanFeatureService) GetPlanFeatures(ctx context.Context, planIDString string) ([]string, error) {
	plan, err := pfs.planRepo.GetByID(ctx, planIDString)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan %s: %w", planIDString, err)
	}

	return plan.FeatureCodes, nil
}

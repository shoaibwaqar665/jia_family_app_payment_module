package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/billing"
	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/cache"
	"github.com/jia-app/paymentservice/internal/shared/events"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// CheckoutUseCase provides business logic for checkout operations
type CheckoutUseCase struct {
	planRepo             repo.PlanRepository
	entitlementRepo      repo.EntitlementRepository
	pricingZoneRepo      repo.PricingZoneRepository
	cache                *cache.Cache // Can be nil if Redis is not available
	entitlementPublisher events.EntitlementPublisher
}

// NewCheckoutUseCase creates a new checkout use case
func NewCheckoutUseCase(
	planRepo repo.PlanRepository,
	entitlementRepo repo.EntitlementRepository,
	pricingZoneRepo repo.PricingZoneRepository,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
) *CheckoutUseCase {
	return &CheckoutUseCase{
		planRepo:             planRepo,
		entitlementRepo:      entitlementRepo,
		pricingZoneRepo:      pricingZoneRepo,
		cache:                cache,
		entitlementPublisher: entitlementPublisher,
	}
}

// CreateCheckoutSession creates a checkout session for a plan
func (uc *CheckoutUseCase) CreateCheckoutSession(ctx context.Context, planID, userID string, familyID *string, countryCode string) (*CheckoutSessionResponse, error) {
	// Validate input
	if planID == "" {
		return nil, status.Error(codes.InvalidArgument, "plan_id is required")
	}
	if userID == "" {
		if contextUserID := extractUserIDFromContext(ctx); contextUserID != "" {
			userID = contextUserID
		} else {
			return nil, status.Error(codes.InvalidArgument, "user_id is required")
		}
	}

	// Validate plan exists
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get plan: %v", err)
	}
	if plan.ID.String() == "" {
		return nil, status.Error(codes.NotFound, "plan not found")
	}

	// Calculate pricing based on country code
	basePrice := plan.PriceCents // Plan price in cents
	adjustedPrice := basePrice
	pricingMultiplier := 1.0

	if countryCode != "" {
		// Get pricing zone for the country
		pricingZone, err := uc.pricingZoneRepo.GetByISOCode(ctx, strings.ToUpper(countryCode))
		if err == nil {
			adjustedPrice = pricingZone.CalculateAdjustedPrice(basePrice)
			pricingMultiplier = pricingZone.PricingMultiplier

			log.Info(ctx, "Applied dynamic pricing",
				zap.String("country_code", countryCode),
				zap.String("zone", pricingZone.Zone),
				zap.String("zone_name", pricingZone.ZoneName),
				zap.Float64("multiplier", pricingMultiplier),
				zap.Int64("base_price", basePrice),
				zap.Int64("adjusted_price", adjustedPrice))
		} else {
			log.Warn(ctx, "Pricing zone not found, using base price",
				zap.String("country_code", countryCode),
				zap.Error(err))
		}
	}

	// Generate placeholder session
	sessionID := fmt.Sprintf("sess_%s", uuid.New().String()[:8])
	redirectURL := fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID)

	log.Info(ctx, "Creating checkout session",
		zap.String("plan_id", planID),
		zap.String("user_id", userID),
		zap.String("family_id", getStringValue(familyID)),
		zap.String("country_code", countryCode),
		zap.Int64("base_price", basePrice),
		zap.Int64("adjusted_price", adjustedPrice),
		zap.Float64("pricing_multiplier", pricingMultiplier),
		zap.String("provider", "stripe"))

	return &CheckoutSessionResponse{
		Provider:          "stripe",
		SessionID:         sessionID,
		RedirectURL:       redirectURL,
		BasePrice:         basePrice,
		AdjustedPrice:     adjustedPrice,
		PricingMultiplier: pricingMultiplier,
	}, nil
}

// ApplyWebhook applies a webhook result from billing provider
func (uc *CheckoutUseCase) ApplyWebhook(ctx context.Context, wr billing.WebhookResult) error {
	// Validate webhook result
	if wr.UserID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required in webhook result")
	}
	if wr.FeatureCode == "" {
		return status.Error(codes.InvalidArgument, "feature_code is required in webhook result")
	}
	if wr.PlanID == uuid.Nil {
		return status.Error(codes.InvalidArgument, "plan_id is required in webhook result")
	}

	// Create or update entitlement
	// Use the string plan ID if available, otherwise use the UUID
	planID := wr.PlanID
	if wr.PlanIDString != "" {
		// Convert string plan ID to UUID for domain model
		planID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(wr.PlanIDString))
	}

	entitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      wr.UserID,
		FamilyID:    wr.FamilyID,
		FeatureCode: wr.FeatureCode,
		PlanID:      planID,
		Status:      "active",
		GrantedAt:   time.Now(),
		ExpiresAt:   wr.ExpiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Upsert entitlement
	savedEntitlement, err := uc.entitlementRepo.Insert(ctx, entitlement)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to upsert entitlement: %v", err)
	}

	// Publish entitlement.updated event
	if uc.entitlementPublisher != nil {
		if err := uc.entitlementPublisher.PublishEntitlementUpdated(ctx, savedEntitlement, "webhook_created"); err != nil {
			log.Error(ctx, "Failed to publish entitlement.updated event", zap.Error(err))
		}
	}

	// Evict cache
	if uc.cache != nil {
		uc.cache.DeleteEntitlement(ctx, savedEntitlement.UserID, savedEntitlement.FeatureCode)
	}

	log.Info(ctx, "Webhook applied successfully",
		zap.String("user_id", wr.UserID),
		zap.String("feature_code", wr.FeatureCode),
		zap.String("plan_id", wr.PlanID.String()))

	return nil
}

// Helper types for responses
type CheckoutSessionResponse struct {
	Provider          string  `json:"provider"`
	SessionID         string  `json:"session_id"`
	RedirectURL       string  `json:"redirect_url"`
	BasePrice         int64   `json:"base_price"`         // Base price in cents
	AdjustedPrice     int64   `json:"adjusted_price"`     // Price after multiplier in cents
	PricingMultiplier float64 `json:"pricing_multiplier"` // Applied multiplier
}

// Helper functions
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

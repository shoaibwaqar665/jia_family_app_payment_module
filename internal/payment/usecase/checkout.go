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
	paymentRepo          repo.PaymentRepository
	cache                *cache.Cache // Can be nil if Redis is not available
	entitlementPublisher events.EntitlementPublisher
	planFeatureService   *PlanFeatureService
}

// NewCheckoutUseCase creates a new checkout use case
func NewCheckoutUseCase(
	planRepo repo.PlanRepository,
	entitlementRepo repo.EntitlementRepository,
	pricingZoneRepo repo.PricingZoneRepository,
	paymentRepo repo.PaymentRepository,
	cache *cache.Cache,
	entitlementPublisher events.EntitlementPublisher,
) *CheckoutUseCase {
	planFeatureService := NewPlanFeatureService(planRepo)
	return &CheckoutUseCase{
		planRepo:             planRepo,
		entitlementRepo:      entitlementRepo,
		pricingZoneRepo:      pricingZoneRepo,
		paymentRepo:          paymentRepo,
		cache:                cache,
		entitlementPublisher: entitlementPublisher,
		planFeatureService:   planFeatureService,
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
	basePrice := plan.PriceDollars // Plan price in dollars
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
				zap.Float64("base_price", basePrice),
				zap.Float64("adjusted_price", adjustedPrice))
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
		zap.Float64("base_price", basePrice),
		zap.Float64("adjusted_price", adjustedPrice),
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
	if wr.PlanIDString == "" {
		return status.Error(codes.InvalidArgument, "plan_id_string is required in webhook result")
	}

	// Grant all entitlements for the plan
	grantedFeatures, err := uc.planFeatureService.GrantEntitlementsForPlan(
		ctx,
		wr.UserID,
		wr.PlanIDString,
		wr.FamilyID,
		&wr.SubscriptionID,
		wr.ExpiresAt,
	)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to grant entitlements for plan %s: %v", wr.PlanIDString, err)
	}

	// Create entitlements for each feature
	for _, featureCode := range grantedFeatures {
		// Convert string plan ID to UUID for domain model
		planID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(wr.PlanIDString))

		// Check if entitlement already exists
		existingEntitlement, found, err := uc.entitlementRepo.Check(ctx, wr.UserID, featureCode)
		if err != nil {
			log.Error(ctx, "Failed to check existing entitlement",
				zap.String("user_id", wr.UserID),
				zap.String("feature_code", featureCode),
				zap.Error(err))
			continue
		}

		// If entitlement already exists and is active, skip creation
		if found && existingEntitlement.Status == "active" {
			log.Info(ctx, "Entitlement already exists, skipping creation",
				zap.String("user_id", wr.UserID),
				zap.String("feature_code", featureCode),
				zap.String("plan_id", wr.PlanIDString))
			continue
		}

		entitlement := domain.Entitlement{
			ID:          uuid.New(),
			UserID:      wr.UserID,
			FamilyID:    wr.FamilyID,
			FeatureCode: featureCode,
			PlanID:      planID,
			Status:      "active",
			GrantedAt:   time.Now(),
			ExpiresAt:   wr.ExpiresAt,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Set subscription ID if provided
		if wr.SubscriptionID != "" {
			entitlement.SubscriptionID = &wr.SubscriptionID
		}

		// Insert entitlement
		savedEntitlement, err := uc.entitlementRepo.Insert(ctx, entitlement)
		if err != nil {
			log.Error(ctx, "Failed to insert entitlement",
				zap.String("user_id", wr.UserID),
				zap.String("feature_code", featureCode),
				zap.String("plan_id", wr.PlanIDString),
				zap.Error(err))
			continue // Continue with other entitlements even if one fails
		}

		// Publish entitlement.updated event for each entitlement
		if uc.entitlementPublisher != nil {
			if err := uc.entitlementPublisher.PublishEntitlementUpdated(ctx, savedEntitlement, "webhook_created"); err != nil {
				log.Error(ctx, "Failed to publish entitlement.updated event", zap.Error(err))
			}
		}

		// Evict cache for this entitlement
		if uc.cache != nil {
			uc.cache.DeleteEntitlement(ctx, savedEntitlement.UserID, savedEntitlement.FeatureCode)
		}

		log.Info(ctx, "Entitlement created successfully",
			zap.String("user_id", wr.UserID),
			zap.String("feature_code", featureCode),
			zap.String("plan_id", wr.PlanIDString),
			zap.String("family_id", getStringValue(wr.FamilyID)))
	}

	// Update payment status to completed if session ID is provided
	if wr.SessionID != "" {
		log.Info(ctx, "Attempting to update payment status",
			zap.String("session_id", wr.SessionID),
			zap.String("user_id", wr.UserID),
			zap.String("plan_id", wr.PlanIDString))

		// Find payment by order ID (session ID)
		payment, err := uc.paymentRepo.GetByOrderID(ctx, wr.SessionID)
		if err == nil && payment != nil {
			log.Info(ctx, "Found payment for session ID",
				zap.String("payment_id", payment.ID.String()),
				zap.String("session_id", wr.SessionID),
				zap.String("current_status", payment.Status))

			// Update payment status to completed
			if err := uc.paymentRepo.UpdateStatus(ctx, payment.ID.String(), "completed"); err != nil {
				log.Error(ctx, "Failed to update payment status",
					zap.String("payment_id", payment.ID.String()),
					zap.String("session_id", wr.SessionID),
					zap.Error(err))
			} else {
				log.Info(ctx, "Payment status updated to completed",
					zap.String("payment_id", payment.ID.String()),
					zap.String("session_id", wr.SessionID))
			}
		} else {
			log.Warn(ctx, "Payment not found for session ID",
				zap.String("session_id", wr.SessionID),
				zap.Error(err))
		}
	} else {
		log.Warn(ctx, "No session ID provided in webhook result",
			zap.String("user_id", wr.UserID),
			zap.String("plan_id", wr.PlanIDString))
	}

	log.Info(ctx, "Webhook applied successfully",
		zap.String("user_id", wr.UserID),
		zap.String("plan_id", wr.PlanIDString),
		zap.String("granted_features", strings.Join(grantedFeatures, ",")),
		zap.String("family_id", getStringValue(wr.FamilyID)))

	return nil
}

// Helper types for responses
type CheckoutSessionResponse struct {
	Provider          string  `json:"provider"`
	SessionID         string  `json:"session_id"`
	RedirectURL       string  `json:"redirect_url"`
	BasePrice         float64 `json:"base_price"`         // Base price in dollars
	AdjustedPrice     float64 `json:"adjusted_price"`     // Price after multiplier in dollars
	PricingMultiplier float64 `json:"pricing_multiplier"` // Applied multiplier
}

// Helper functions
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

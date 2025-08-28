package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// CheckoutUseCase provides business logic for checkout operations
type CheckoutUseCase struct {
	planRepo PlanRepository
}

// PlanRepository defines the interface for plan operations
type PlanRepository interface {
	GetByID(ctx context.Context, id string) (domain.Plan, error)
	ListActive(ctx context.Context) ([]domain.Plan, error)
}

// NewCheckoutUseCase creates a new checkout use case
func NewCheckoutUseCase(planRepo PlanRepository) *CheckoutUseCase {
	return &CheckoutUseCase{
		planRepo: planRepo,
	}
}

// CreateCheckoutSession creates a checkout session for a plan
func (uc *CheckoutUseCase) CreateCheckoutSession(ctx context.Context, planID, userID string) (*CheckoutSessionResponse, error) {
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

	// Generate placeholder session
	sessionID := fmt.Sprintf("sess_%s", uuid.New().String()[:8])
	redirectURL := fmt.Sprintf("https://checkout.stripe.com/pay/%s", sessionID)

	log.Info(ctx, "TODO: integrate with actual payment provider",
		zap.String("plan_id", planID),
		zap.String("user_id", userID),
		zap.String("provider", "stripe"))

	return &CheckoutSessionResponse{
		Provider:    "stripe",
		SessionID:   sessionID,
		RedirectURL: redirectURL,
	}, nil
}

// Helper types for responses
type CheckoutSessionResponse struct {
	Provider    string `json:"provider"`
	SessionID   string `json:"session_id"`
	RedirectURL string `json:"redirect_url"`
}

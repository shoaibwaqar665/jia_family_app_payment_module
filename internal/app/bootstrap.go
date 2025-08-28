package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/billing"
	"github.com/jia-app/paymentservice/internal/billing/stripebp"
	"github.com/jia-app/paymentservice/internal/shared/config"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// NewBillingProvider creates a billing provider based on configuration
func NewBillingProvider(ctx context.Context, cfg *config.Config) (billing.Provider, error) {
	logger := log.L(ctx)

	log.Info(ctx, "Initializing billing provider",
		zap.String("provider", cfg.Billing.Provider))

	switch cfg.Billing.Provider {
	case "stripe":
		return NewStripeProvider(ctx, cfg, logger)
	case "mock", "noop":
		return NewMockProvider(ctx, logger)
	default:
		return nil, fmt.Errorf("unsupported billing provider: %s", cfg.Billing.Provider)
	}
}

// NewStripeProvider creates a Stripe billing provider
func NewStripeProvider(ctx context.Context, cfg *config.Config, logger *zap.Logger) (billing.Provider, error) {
	if cfg.Billing.StripeSecret == "" {
		return nil, fmt.Errorf("stripe secret key is required")
	}

	if cfg.Billing.StripePublishable == "" {
		log.Warn(ctx, "Stripe publishable key not configured - some features may not work")
	}

	provider := stripebp.NewAdapter(
		cfg.Billing.StripeSecret,
		cfg.Billing.StripePublishable,
		logger,
	)

	log.Info(ctx, "Stripe billing provider initialized successfully",
		zap.String("publishable_key_prefix", getKeyPrefix(cfg.Billing.StripePublishable)))

	return provider, nil
}

// NewMockProvider creates a mock billing provider for testing/development
func NewMockProvider(ctx context.Context, logger *zap.Logger) (billing.Provider, error) {
	log.Info(ctx, "Using mock billing provider for testing/development")

	// Return a mock implementation that logs all operations
	return &MockProvider{logger: logger}, nil
}

// getKeyPrefix returns the first 8 characters of a key for logging (for security)
func getKeyPrefix(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "***"
}

// MockProvider is a mock implementation of billing.Provider for testing
type MockProvider struct {
	logger *zap.Logger
}

// CreateCheckoutSession creates a mock checkout session
func (m *MockProvider) CreateCheckoutSession(ctx context.Context, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error) {
	m.logger.Info("Mock: Creating checkout session",
		zap.String("plan_id", req.PlanID.String()),
		zap.String("user_id", req.UserID))

	return &billing.CreateCheckoutSessionResponse{
		SessionID: "mock_session_" + req.PlanID.String()[:8],
		URL:       "https://mock-checkout.example.com/session",
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours from now
	}, nil
}

// GetSession retrieves a mock session
func (m *MockProvider) GetSession(ctx context.Context, sessionID string) (*billing.Session, error) {
	m.logger.Info("Mock: Getting session", zap.String("session_id", sessionID))

	now := time.Now()
	return &billing.Session{
		ID:        sessionID,
		Status:    string(billing.SessionStatusOpen),
		URL:       "https://mock-checkout.example.com/session",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// CancelSession cancels a mock session
func (m *MockProvider) CancelSession(ctx context.Context, sessionID string) error {
	m.logger.Info("Mock: Cancelling session", zap.String("session_id", sessionID))
	return nil
}

// ValidateWebhook validates a mock webhook
func (m *MockProvider) ValidateWebhook(ctx context.Context, payload []byte, signature string) error {
	m.logger.Info("Mock: Validating webhook", zap.String("signature", signature))
	return nil
}

// ParseWebhook parses a mock webhook
func (m *MockProvider) ParseWebhook(ctx context.Context, payload []byte) (*billing.WebhookResult, error) {
	m.logger.Info("Mock: Parsing webhook", zap.Int("payload_size", len(payload)))

	return &billing.WebhookResult{
		EventType:   string(billing.WebhookEventTypeCheckoutSessionCompleted),
		SessionID:   "mock_session_123",
		UserID:      "spiff_id_mock_user",
		FeatureCode: "premium_feature",
		PlanID:      uuid.New(), // Generate a new UUID for mock
		Amount:      2999,
		Currency:    "usd",
		Status:      "succeeded",
		ExpiresAt:   nil,
		Metadata: map[string]interface{}{
			"mock": true,
		},
	}, nil
}

// Close closes the mock provider
func (m *MockProvider) Close() error {
	m.logger.Info("Mock: Closing provider")
	return nil
}

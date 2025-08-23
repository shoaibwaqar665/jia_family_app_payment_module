package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/jia-app/paymentservice/internal/domain"
	"github.com/jia-app/paymentservice/internal/log"
)

// TestCheckEntitlement tests the CheckEntitlement method
func TestCheckEntitlement(t *testing.T) {
	// Initialize logger for tests
	_ = log.Init("info")

	// Create service with nil dependencies for basic testing
	service := &PaymentService{}

	ctx := context.Background()

	// Test invalid input - empty user ID and feature code
	_, err := service.CheckEntitlement(ctx, "", "")
	if err == nil {
		t.Error("Expected error for empty user_id")
	}

	// Check if it's the right error code
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument error, got: %v", err)
	}

	// Test invalid input - empty feature code
	_, err = service.CheckEntitlement(ctx, "user123", "")
	if err == nil {
		t.Error("Expected error for empty feature_code")
	}

	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument error, got: %v", err)
	}
}

// TestListUserEntitlements tests the ListUserEntitlements method
func TestListUserEntitlements(t *testing.T) {
	// Initialize logger for tests
	_ = log.Init("info")

	// Create service with nil dependencies for basic testing
	service := &PaymentService{}

	ctx := context.Background()

	// Test invalid input - empty user ID
	_, err := service.ListUserEntitlements(ctx, "")
	if err == nil {
		t.Error("Expected error for empty user_id")
	}

	// Check if it's the right error code
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument error, got: %v", err)
	}
}

// TestCreateCheckoutSession tests the CreateCheckoutSession method
func TestCreateCheckoutSession(t *testing.T) {
	// Initialize logger for tests
	_ = log.Init("info")

	// Create service with nil dependencies for basic testing
	service := &PaymentService{}

	ctx := context.Background()

	// Test invalid input - empty plan ID
	_, err := service.CreateCheckoutSession(ctx, "", "user123")
	if err == nil {
		t.Error("Expected error for empty plan_id")
	}

	// Check if it's the right error code
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument error, got: %v", err)
	}

	// Test invalid input - empty user ID
	_, err = service.CreateCheckoutSession(ctx, "plan123", "")
	if err == nil {
		t.Error("Expected error for empty user_id")
	}

	st, ok = status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument error, got: %v", err)
	}
}

// TestPaymentSuccessWebhook tests the PaymentSuccessWebhook method
func TestPaymentSuccessWebhook(t *testing.T) {
	// Initialize logger for tests
	_ = log.Init("info")

	// Create service with nil dependencies for basic testing
	service := &PaymentService{}

	ctx := context.Background()

	// Test invalid signature
	err := service.PaymentSuccessWebhook(ctx, []byte("test payload"), "")
	if err == nil {
		t.Error("Expected error for empty signature")
	}

	// Check if it's the right error code
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated error, got: %v", err)
	}
}

// TestIsValidEntitlement tests the isValidEntitlement helper function
func TestIsValidEntitlement(t *testing.T) {
	// Test nil entitlement
	if isValidEntitlement(nil) {
		t.Error("Expected false for nil entitlement")
	}

	// Test inactive entitlement
	inactive := &domain.Entitlement{
		Status: "inactive",
	}
	if isValidEntitlement(inactive) {
		t.Error("Expected false for inactive entitlement")
	}

	// Test expired entitlement
	pastTime := time.Now().Add(-1 * time.Hour)
	expired := &domain.Entitlement{
		Status:    "active",
		ExpiresAt: &pastTime,
	}
	if isValidEntitlement(expired) {
		t.Error("Expected false for expired entitlement")
	}

	// Test valid entitlement (active, not expired)
	valid := &domain.Entitlement{
		Status:    "active",
		ExpiresAt: nil, // Never expires
	}
	if !isValidEntitlement(valid) {
		t.Error("Expected true for valid entitlement")
	}

	// Test valid entitlement (active, expires in future)
	futureTime := time.Now().Add(1 * time.Hour)
	validFuture := &domain.Entitlement{
		Status:    "active",
		ExpiresAt: &futureTime,
	}
	if !isValidEntitlement(validFuture) {
		t.Error("Expected true for valid future-expiring entitlement")
	}
}

// TestExtractUserIDFromContext tests the extractUserIDFromContext helper function
func TestExtractUserIDFromContext(t *testing.T) {
	// Test empty context
	ctx := context.Background()
	userID := extractUserIDFromContext(ctx)
	if userID != "" {
		t.Errorf("Expected empty user ID, got: %s", userID)
	}

	// Test context with user ID
	ctxWithUser := context.WithValue(ctx, log.UserIDKey, "user123")
	userID = extractUserIDFromContext(ctxWithUser)
	if userID != "user123" {
		t.Errorf("Expected user123, got: %s", userID)
	}

	// Test context with non-string user ID
	ctxWithInvalidUser := context.WithValue(ctx, log.UserIDKey, 123)
	userID = extractUserIDFromContext(ctxWithInvalidUser)
	if userID != "" {
		t.Errorf("Expected empty user ID for non-string value, got: %s", userID)
	}
}

// TestValidateWebhookSignature tests the validateWebhookSignature method
func TestValidateWebhookSignature(t *testing.T) {
	service := &PaymentService{}

	// Test empty signature
	err := service.validateWebhookSignature([]byte("payload"), "")
	if err == nil {
		t.Error("Expected error for empty signature")
	}

	// Test non-empty signature (should pass with current stub implementation)
	err = service.validateWebhookSignature([]byte("payload"), "valid_signature")
	if err != nil {
		t.Errorf("Expected no error for valid signature, got: %v", err)
	}
}

// TestParseWebhookPayload tests the parseWebhookPayload method
func TestParseWebhookPayload(t *testing.T) {
	service := &PaymentService{}

	// Test parsing (stub implementation should always succeed)
	data, err := service.parseWebhookPayload([]byte("test payload"))
	if err != nil {
		t.Errorf("Expected no error from stub implementation, got: %v", err)
	}

	if data == nil {
		t.Error("Expected webhook data to be returned")
	}

	if data.UserID == "" {
		t.Error("Expected UserID to be set in webhook data")
	}

	if data.FeatureCode == "" {
		t.Error("Expected FeatureCode to be set in webhook data")
	}
}

// TestHelperTypes tests the helper response types
func TestHelperTypes(t *testing.T) {
	// Test CheckEntitlementResponse
	entitlement := &domain.Entitlement{
		ID:          uuid.New(),
		UserID:      "user123",
		FeatureCode: "premium",
		Status:      "active",
	}

	response := &CheckEntitlementResponse{
		Allowed:     true,
		Entitlement: entitlement,
	}

	if !response.Allowed {
		t.Error("Expected Allowed to be true")
	}

	if response.Entitlement.UserID != "user123" {
		t.Error("Expected entitlement user ID to be user123")
	}

	// Test CheckoutSessionResponse
	checkoutResponse := &CheckoutSessionResponse{
		Provider:    "stripe",
		SessionID:   "sess_123",
		RedirectURL: "https://example.com",
	}

	if checkoutResponse.Provider != "stripe" {
		t.Error("Expected provider to be stripe")
	}

	if checkoutResponse.SessionID != "sess_123" {
		t.Error("Expected session ID to be sess_123")
	}

	// Test WebhookData
	webhookData := &WebhookData{
		UserID:      "user456",
		FeatureCode: "premium",
		PlanID:      uuid.New(),
		ExpiresAt:   nil,
	}

	if webhookData.UserID != "user456" {
		t.Error("Expected UserID to be user456")
	}

	if webhookData.FeatureCode != "premium" {
		t.Error("Expected FeatureCode to be premium")
	}
}

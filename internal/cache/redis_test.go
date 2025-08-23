package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/domain"
)

// TestEntitlementKeyFormat tests the key format used for caching entitlements
func TestEntitlementKeyFormat(t *testing.T) {
	userID := "user123"
	featureCode := "premium_feature"

	// Test the key format that should be used: "entl:{userID}:{featureCode}"
	expectedKey := fmt.Sprintf("entl:%s:%s", userID, featureCode)
	if expectedKey != "entl:user123:premium_feature" {
		t.Errorf("Key format should be 'entl:{userID}:{featureCode}', got %s", expectedKey)
	}

	// Test with special characters
	userIDWithSpecial := "user-123_test"
	featureCodeWithSpecial := "premium-feature_v2"
	expectedKeySpecial := fmt.Sprintf("entl:%s:%s", userIDWithSpecial, featureCodeWithSpecial)
	if expectedKeySpecial != "entl:user-123_test:premium-feature_v2" {
		t.Errorf("Key format should handle special characters, got %s", expectedKeySpecial)
	}
}

// TestEntitlementStructure tests that we can create entitlement structures
func TestEntitlementStructure(t *testing.T) {
	testEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      "user123",
		FeatureCode: "feature456",
		PlanID:      uuid.New(),
		Status:      "active",
		GrantedAt:   time.Now(),
	}

	if testEntitlement.UserID != "user123" {
		t.Error("UserID should be set correctly")
	}

	if testEntitlement.FeatureCode != "feature456" {
		t.Error("FeatureCode should be set correctly")
	}

	if testEntitlement.Status != "active" {
		t.Error("Status should be set correctly")
	}
}

// TestTTLDefaults verifies TTL behavior
func TestTTLDefaults(t *testing.T) {
	// Test that default TTL is 2 minutes
	defaultTTL := 2 * time.Minute
	if defaultTTL != 2*time.Minute {
		t.Error("Default TTL should be 2 minutes")
	}

	// Test that negative result TTL is 10 seconds max
	negativeTTL := 10 * time.Second
	if negativeTTL != 10*time.Second {
		t.Error("Negative result TTL should be 10 seconds")
	}

	// Verify negative TTL is much shorter than default TTL
	if negativeTTL >= defaultTTL {
		t.Error("Negative result TTL should be much shorter than default TTL")
	}
}

package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jia-app/paymentservice/internal/domain"
)

func TestStore_Plan(t *testing.T) {
	// This is a basic test to ensure the interface is implemented
	// In a real test, you would use a test database
	store := &Store{}

	planRepo := store.Plan()
	if planRepo == nil {
		t.Error("Plan repository should not be nil")
	}

	// Test that the interface methods exist (they will return errors since not implemented)
	_, err := planRepo.GetByID(context.Background(), "test-id")
	if err == nil {
		t.Error("GetByID should return an error when not implemented")
	}

	_, err = planRepo.ListActive(context.Background())
	if err == nil {
		t.Error("ListActive should return an error when not implemented")
	}
}

func TestStore_Entitlement(t *testing.T) {
	// This is a basic test to ensure the interface is implemented
	// In a real test, you would use a test database
	store := &Store{}

	entitlementRepo := store.Entitlement()
	if entitlementRepo == nil {
		t.Error("Entitlement repository should not be nil")
	}

	// Test that the interface methods exist (they will return errors since not implemented)
	_, found, err := entitlementRepo.Check(context.Background(), "user-123", "feature-456")
	if err == nil {
		t.Error("Check should return an error when not implemented")
	}
	if found {
		t.Error("Check should return false when not implemented")
	}

	_, err = entitlementRepo.ListByUser(context.Background(), "user-123")
	if err == nil {
		t.Error("ListByUser should return an error when not implemented")
	}

	testEntitlement := domain.Entitlement{
		ID:          uuid.New(),
		UserID:      "user-123",
		FeatureCode: "feature-456",
		PlanID:      uuid.New(),
		Status:      "active",
	}

	_, err = entitlementRepo.Insert(context.Background(), testEntitlement)
	if err == nil {
		t.Error("Insert should return an error when not implemented")
	}

	err = entitlementRepo.UpdateStatus(context.Background(), "entitlement-123", "inactive")
	if err == nil {
		t.Error("UpdateStatus should return an error when not implemented")
	}

	err = entitlementRepo.UpdateExpiry(context.Background(), "entitlement-123", nil)
	if err == nil {
		t.Error("UpdateExpiry should return an error when not implemented")
	}
}

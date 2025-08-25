package pricing

import (
	"testing"
	"time"
)

func TestCalculator_Apply_BasicRules(t *testing.T) {
	store := NewMemoryRuleStore()
	now := time.Now()
	later := now.Add(1 * time.Hour)

	_ = store.Upsert(PricingRule{ID: "loc-us", Type: RuleTypeLocation, Location: "US", Factor: 1.10, Enabled: true})
	_ = store.Upsert(PricingRule{ID: "happy-hour", Type: RuleTypeTime, StartTime: &now, EndTime: &later, Factor: 0.90, Enabled: true})
	_ = store.Upsert(PricingRule{ID: "surge", Type: RuleTypeDemand, DemandThreshold: 50, Factor: 1.20, Enabled: true})

	calc := NewCalculator(store)

	inputs := ContextualInputs{Location: "US", Now: now, Demand: 60}
	// Base 10000 cents -> loc 1.10 -> 11000; time 0.90 -> 9900; demand 1.20 -> 11880
	got := calc.Apply(10000, inputs)
	if got != 11880 {
		t.Fatalf("expected 11880, got %d", got)
	}
}

func TestCalculator_SurgePricing_ClampedAt200(t *testing.T) {
	store := NewMemoryRuleStore()
	// Surge factor that would push above 200%
	_ = store.Upsert(PricingRule{ID: "surge-strong", Type: RuleTypeDemand, DemandThreshold: 1, Factor: 3.0, Enabled: true})
	calc := NewCalculator(store)
	got := calc.Apply(10000, ContextualInputs{Demand: 10})
	if got != 20000 { // 200% clamp
		t.Fatalf("expected 20000, got %d", got)
	}
}

func TestCalculator_Discount_ClampedAt50(t *testing.T) {
	store := NewMemoryRuleStore()
	now := time.Now()
	later := now.Add(1 * time.Hour)
	// Deep discount 0.2 would be 20% of base, should clamp to 50%
	_ = store.Upsert(PricingRule{ID: "deep-discount", Type: RuleTypeTime, StartTime: &now, EndTime: &later, Factor: 0.2, Enabled: true})
	calc := NewCalculator(store)
	got := calc.Apply(10000, ContextualInputs{Now: now})
	if got != 5000 { // 50% clamp
		t.Fatalf("expected 5000, got %d", got)
	}
}

func TestCalculator_LocationFactor_WithinBounds(t *testing.T) {
	store := NewMemoryRuleStore()
	_ = store.Upsert(PricingRule{ID: "loc-uk", Type: RuleTypeLocation, Location: "UK", Factor: 1.5, Enabled: true})
	calc := NewCalculator(store)
	got := calc.Apply(10000, ContextualInputs{Location: "UK"})
	if got != 15000 {
		t.Fatalf("expected 15000, got %d", got)
	}
}

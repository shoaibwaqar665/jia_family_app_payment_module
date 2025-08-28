package payment

import (
	"time"
)

// RuleType represents the kind of pricing rule.
type RuleType string

const (
	RuleTypeLocation RuleType = "location_factor"
	RuleTypeTime     RuleType = "time_discount"
	RuleTypeDemand   RuleType = "demand_surge"
)

// PricingRule defines a single pricing rule configuration.
type PricingRule struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Type RuleType `json:"type"`
	// Factor is used for multiplicative adjustments (e.g., 1.2 for +20%, 0.8 for -20%).
	Factor float64 `json:"factor"`
	// Window defines when the rule is active (optional for time-based rules).
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	// Location applies to location-based rules (e.g., country code, city).
	Location string `json:"location,omitempty"`
	// DemandThreshold is used for surge rules; if current demand >= threshold, apply factor.
	DemandThreshold int `json:"demand_threshold,omitempty"`
	// Enabled flag
	Enabled bool `json:"enabled"`
}

// RuleRepository defines storage for pricing rules. Can be implemented with DB later.
type RuleRepository interface {
	// Get returns a rule by ID. Second return indicates existence.
	Get(id string) (PricingRule, bool)
	// List returns all enabled rules.
	List() []PricingRule
	// Upsert adds or updates a rule.
	Upsert(rule PricingRule) error
	// Delete removes a rule.
	Delete(id string) error
}

// ContextualInputs capture dynamic inputs to the calculator.
type ContextualInputs struct {
	Location string
	Now      time.Time
	Demand   int
}

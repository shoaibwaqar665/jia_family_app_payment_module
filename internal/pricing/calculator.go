package pricing

import (
	"math"
	"time"
)

// Calculator computes adjusted prices from a base price and a set of rules.
type Calculator struct {
	rules RuleRepository
}

func NewCalculator(repo RuleRepository) *Calculator {
	return &Calculator{rules: repo}
}

// Apply computes the adjusted price given a base price and contextual inputs.
// It applies all enabled rules in the repository. Optional: allow selecting by IDs later.
func (c *Calculator) Apply(basePriceCents int64, ctx ContextualInputs) int64 {
	if basePriceCents <= 0 {
		return 0
	}

	price := float64(basePriceCents)
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}

	for _, rule := range c.rules.List() {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case RuleTypeLocation:
			if rule.Location != "" && ctx.Location != "" && rule.Location == ctx.Location {
				price = applyFactor(price, rule.Factor)
			}
		case RuleTypeTime:
			if rule.StartTime != nil && rule.EndTime != nil {
				if now.After(*rule.StartTime) && now.Before(*rule.EndTime) {
					price = applyFactor(price, rule.Factor)
				}
			}
		case RuleTypeDemand:
			if rule.DemandThreshold > 0 && ctx.Demand >= rule.DemandThreshold {
				price = applyFactor(price, rule.Factor)
			}
		default:
			// Unknown rule types are ignored
		}
	}

	// clamp to 50%-200% of base
	min := float64(basePriceCents) * 0.5
	max := float64(basePriceCents) * 2.0
	if price < min {
		price = min
	}
	if price > max {
		price = max
	}

	// round to nearest cent integer
	return int64(math.Round(price))
}

// ApplyWithRules computes adjusted price using provided rules instead of repository list.
func (c *Calculator) ApplyWithRules(basePriceCents int64, ctx ContextualInputs, rules []PricingRule) int64 {
	if basePriceCents <= 0 {
		return 0
	}

	price := float64(basePriceCents)
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		switch rule.Type {
		case RuleTypeLocation:
			if rule.Location != "" && ctx.Location != "" && rule.Location == ctx.Location {
				price = applyFactor(price, rule.Factor)
			}
		case RuleTypeTime:
			if rule.StartTime != nil && rule.EndTime != nil {
				if now.After(*rule.StartTime) && now.Before(*rule.EndTime) {
					price = applyFactor(price, rule.Factor)
				}
			}
		case RuleTypeDemand:
			if rule.DemandThreshold > 0 && ctx.Demand >= rule.DemandThreshold {
				price = applyFactor(price, rule.Factor)
			}
		default:
			// ignore
		}
	}

	// clamp to 50%-200% of base
	min := float64(basePriceCents) * 0.5
	max := float64(basePriceCents) * 2.0
	if price < min {
		price = min
	}
	if price > max {
		price = max
	}

	return int64(math.Round(price))
}

func applyFactor(price float64, factor float64) float64 {
	if factor == 0 {
		return price
	}
	return price * factor
}

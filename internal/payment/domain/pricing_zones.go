package domain

import (
	"time"
)

// PricingZone represents a pricing zone for dynamic pricing
type PricingZone struct {
	ID                        string    `json:"id" db:"id"`
	Country                   string    `json:"country" db:"country"`
	ISOCode                   string    `json:"iso_code" db:"iso_code"`
	Zone                      string    `json:"zone" db:"zone"`
	ZoneName                  string    `json:"zone_name" db:"zone_name"`
	WorldBankClassification   string    `json:"world_bank_classification" db:"world_bank_classification"`
	GNIPerCapitaThreshold     string    `json:"gni_per_capita_threshold" db:"gni_per_capita_threshold"`
	PricingMultiplier         float64   `json:"pricing_multiplier" db:"pricing_multiplier"`
	CreatedAt                 time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at" db:"updated_at"`
}

// PricingZoneRequest represents a request for pricing zone operations
type PricingZoneRequest struct {
	Country   string  `json:"country,omitempty"`
	ISOCode   string  `json:"iso_code,omitempty"`
	Zone      string  `json:"zone,omitempty"`
	BasePrice int64   `json:"base_price"` // Base price in cents
}

// PricingZoneResponse represents the response from pricing zone operations
type PricingZoneResponse struct {
	Zone              *PricingZone `json:"zone,omitempty"`
	BasePrice         int64        `json:"base_price"`         // Original base price in cents
	AdjustedPrice     int64        `json:"adjusted_price"`     // Price after multiplier in cents
	PricingMultiplier float64      `json:"pricing_multiplier"` // Applied multiplier
	Currency          string       `json:"currency"`           // Currency code
}

// ZoneType represents the type of pricing zone
type ZoneType string

const (
	ZoneTypePremium  ZoneType = "A" // Premium - 100% of base price
	ZoneTypeMidHigh  ZoneType = "B" // Mid-High - 70% of base price
	ZoneTypeMidLow   ZoneType = "C" // Mid-Low - 40% of base price
	ZoneTypeLowIncome ZoneType = "D" // Low-Income - 20% of base price
)

// GetZoneType returns the ZoneType for a given zone string
func GetZoneType(zone string) ZoneType {
	switch zone {
	case "A":
		return ZoneTypePremium
	case "B":
		return ZoneTypeMidHigh
	case "C":
		return ZoneTypeMidLow
	case "D":
		return ZoneTypeLowIncome
	default:
		return ZoneTypePremium // Default to premium pricing
	}
}

// GetDefaultMultiplier returns the default pricing multiplier for a zone type
func GetDefaultMultiplier(zoneType ZoneType) float64 {
	switch zoneType {
	case ZoneTypePremium:
		return 1.00
	case ZoneTypeMidHigh:
		return 0.70
	case ZoneTypeMidLow:
		return 0.40
	case ZoneTypeLowIncome:
		return 0.20
	default:
		return 1.00
	}
}

// CalculateAdjustedPrice calculates the adjusted price based on the pricing multiplier
func (pz *PricingZone) CalculateAdjustedPrice(basePrice int64) int64 {
	if pz.PricingMultiplier <= 0 {
		return basePrice
	}
	
	adjustedPrice := float64(basePrice) * pz.PricingMultiplier
	return int64(adjustedPrice)
}

// IsValidZone checks if the zone is valid
func IsValidZone(zone string) bool {
	switch zone {
	case "A", "B", "C", "D":
		return true
	default:
		return false
	}
}

// GetZoneName returns the human-readable name for a zone
func GetZoneName(zone string) string {
	switch zone {
	case "A":
		return "Premium"
	case "B":
		return "Mid-High"
	case "C":
		return "Mid-Low"
	case "D":
		return "Low-Income"
	default:
		return "Unknown"
	}
}

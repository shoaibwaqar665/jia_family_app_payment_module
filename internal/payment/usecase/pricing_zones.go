package usecase

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/jia-app/paymentservice/internal/payment/domain"
	"github.com/jia-app/paymentservice/internal/payment/repo"
	"github.com/jia-app/paymentservice/internal/shared/log"
)

// PricingZoneUseCase provides business logic for pricing zone operations
type PricingZoneUseCase struct {
	pricingZoneRepo repo.PricingZoneRepository
}

// NewPricingZoneUseCase creates a new pricing zone use case
func NewPricingZoneUseCase(pricingZoneRepo repo.PricingZoneRepository) *PricingZoneUseCase {
	return &PricingZoneUseCase{
		pricingZoneRepo: pricingZoneRepo,
	}
}

// GetPricingZoneByISOCode retrieves a pricing zone by ISO country code
func (uc *PricingZoneUseCase) GetPricingZoneByISOCode(ctx context.Context, isoCode string) (*domain.PricingZone, error) {
	// Normalize ISO code to uppercase
	isoCode = strings.ToUpper(strings.TrimSpace(isoCode))

	if isoCode == "" {
		return nil, fmt.Errorf("ISO code is required")
	}

	zone, err := uc.pricingZoneRepo.GetByISOCode(ctx, isoCode)
	if err != nil {
		log.Error(ctx, "Failed to get pricing zone by ISO code",
			zap.String("iso_code", isoCode),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get pricing zone: %w", err)
	}

	return &zone, nil
}

// GetPricingZoneByCountry retrieves a pricing zone by country name
func (uc *PricingZoneUseCase) GetPricingZoneByCountry(ctx context.Context, country string) (*domain.PricingZone, error) {
	country = strings.TrimSpace(country)

	if country == "" {
		return nil, fmt.Errorf("country name is required")
	}

	zone, err := uc.pricingZoneRepo.GetByCountry(ctx, country)
	if err != nil {
		log.Error(ctx, "Failed to get pricing zone by country",
			zap.String("country", country),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get pricing zone: %w", err)
	}

	return &zone, nil
}

// CalculateAdjustedPrice calculates the adjusted price for a given country/ISO code
func (uc *PricingZoneUseCase) CalculateAdjustedPrice(ctx context.Context, req domain.PricingZoneRequest) (*domain.PricingZoneResponse, error) {
	var zone domain.PricingZone
	var err error

	// Try to get pricing zone by ISO code first, then by country
	if req.ISOCode != "" {
		zone, err = uc.pricingZoneRepo.GetByISOCode(ctx, strings.ToUpper(req.ISOCode))
	} else if req.Country != "" {
		zone, err = uc.pricingZoneRepo.GetByCountry(ctx, req.Country)
	} else {
		return nil, fmt.Errorf("either ISO code or country name is required")
	}

	if err != nil {
		log.Warn(ctx, "Pricing zone not found, using default premium pricing",
			zap.String("iso_code", req.ISOCode),
			zap.String("country", req.Country),
			zap.Error(err))

		// Use default premium pricing if zone not found
		zone = domain.PricingZone{
			Zone:              "A",
			ZoneName:          "Premium",
			PricingMultiplier: 1.00,
		}
	}

	// Calculate adjusted price
	adjustedPrice := zone.CalculateAdjustedPrice(req.BasePrice)

	response := &domain.PricingZoneResponse{
		Zone:              &zone,
		BasePrice:         req.BasePrice,
		AdjustedPrice:     adjustedPrice,
		PricingMultiplier: zone.PricingMultiplier,
		Currency:          "USD", // Default currency, could be made configurable
	}

	log.Info(ctx, "Price calculated successfully",
		zap.String("zone", zone.Zone),
		zap.String("zone_name", zone.ZoneName),
		zap.Float64("multiplier", zone.PricingMultiplier),
		zap.Float64("base_price", req.BasePrice),
		zap.Float64("adjusted_price", adjustedPrice))

	return response, nil
}

// ListPricingZones retrieves all pricing zones
func (uc *PricingZoneUseCase) ListPricingZones(ctx context.Context) ([]domain.PricingZone, error) {
	zones, err := uc.pricingZoneRepo.List(ctx)
	if err != nil {
		log.Error(ctx, "Failed to list pricing zones", zap.Error(err))
		return nil, fmt.Errorf("failed to list pricing zones: %w", err)
	}

	return zones, nil
}

// GetPricingZonesByZone retrieves all pricing zones for a specific zone type
func (uc *PricingZoneUseCase) GetPricingZonesByZone(ctx context.Context, zone string) ([]domain.PricingZone, error) {
	zone = strings.ToUpper(strings.TrimSpace(zone))

	if !domain.IsValidZone(zone) {
		return nil, fmt.Errorf("invalid zone type: %s", zone)
	}

	zones, err := uc.pricingZoneRepo.GetByZone(ctx, zone)
	if err != nil {
		log.Error(ctx, "Failed to get pricing zones by zone",
			zap.String("zone", zone),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get pricing zones: %w", err)
	}

	return zones, nil
}

// UpsertPricingZone creates or updates a pricing zone
func (uc *PricingZoneUseCase) UpsertPricingZone(ctx context.Context, zone domain.PricingZone) (*domain.PricingZone, error) {
	// Validate zone
	if zone.ISOCode == "" {
		return nil, fmt.Errorf("ISO code is required")
	}
	if zone.Country == "" {
		return nil, fmt.Errorf("country name is required")
	}
	if !domain.IsValidZone(zone.Zone) {
		return nil, fmt.Errorf("invalid zone type: %s", zone.Zone)
	}
	if zone.PricingMultiplier < 0 {
		return nil, fmt.Errorf("pricing multiplier must be non-negative")
	}

	// Normalize data
	zone.ISOCode = strings.ToUpper(strings.TrimSpace(zone.ISOCode))
	zone.Country = strings.TrimSpace(zone.Country)
	zone.Zone = strings.ToUpper(strings.TrimSpace(zone.Zone))

	// Set zone name if not provided
	if zone.ZoneName == "" {
		zone.ZoneName = domain.GetZoneName(zone.Zone)
	}

	savedZone, err := uc.pricingZoneRepo.Upsert(ctx, zone)
	if err != nil {
		log.Error(ctx, "Failed to upsert pricing zone",
			zap.String("iso_code", zone.ISOCode),
			zap.Error(err))
		return nil, fmt.Errorf("failed to upsert pricing zone: %w", err)
	}

	log.Info(ctx, "Pricing zone upserted successfully",
		zap.String("iso_code", savedZone.ISOCode),
		zap.String("country", savedZone.Country),
		zap.String("zone", savedZone.Zone),
		zap.Float64("multiplier", savedZone.PricingMultiplier))

	return &savedZone, nil
}

// BulkUpsertPricingZones creates or updates multiple pricing zones
func (uc *PricingZoneUseCase) BulkUpsertPricingZones(ctx context.Context, zones []domain.PricingZone) error {
	if len(zones) == 0 {
		return fmt.Errorf("no pricing zones provided")
	}

	// Validate all zones
	for i, zone := range zones {
		if zone.ISOCode == "" {
			return fmt.Errorf("ISO code is required for zone at index %d", i)
		}
		if zone.Country == "" {
			return fmt.Errorf("country name is required for zone at index %d", i)
		}
		if !domain.IsValidZone(zone.Zone) {
			return fmt.Errorf("invalid zone type '%s' for zone at index %d", zone.Zone, i)
		}
		if zone.PricingMultiplier < 0 {
			return fmt.Errorf("pricing multiplier must be non-negative for zone at index %d", i)
		}

		// Normalize data
		zones[i].ISOCode = strings.ToUpper(strings.TrimSpace(zone.ISOCode))
		zones[i].Country = strings.TrimSpace(zone.Country)
		zones[i].Zone = strings.ToUpper(strings.TrimSpace(zone.Zone))

		// Set zone name if not provided
		if zones[i].ZoneName == "" {
			zones[i].ZoneName = domain.GetZoneName(zones[i].Zone)
		}
	}

	err := uc.pricingZoneRepo.BulkUpsert(ctx, zones)
	if err != nil {
		log.Error(ctx, "Failed to bulk upsert pricing zones",
			zap.Int("count", len(zones)),
			zap.Error(err))
		return fmt.Errorf("failed to bulk upsert pricing zones: %w", err)
	}

	log.Info(ctx, "Pricing zones bulk upserted successfully",
		zap.Int("count", len(zones)))

	return nil
}

// DeletePricingZone deletes a pricing zone by ISO code
func (uc *PricingZoneUseCase) DeletePricingZone(ctx context.Context, isoCode string) error {
	isoCode = strings.ToUpper(strings.TrimSpace(isoCode))

	if isoCode == "" {
		return fmt.Errorf("ISO code is required")
	}

	err := uc.pricingZoneRepo.Delete(ctx, isoCode)
	if err != nil {
		log.Error(ctx, "Failed to delete pricing zone",
			zap.String("iso_code", isoCode),
			zap.Error(err))
		return fmt.Errorf("failed to delete pricing zone: %w", err)
	}

	log.Info(ctx, "Pricing zone deleted successfully",
		zap.String("iso_code", isoCode))

	return nil
}

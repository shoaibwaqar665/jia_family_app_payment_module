-- name: GetPricingZoneByISOCode :one
SELECT id, country, iso_code, zone, zone_name, world_bank_classification, 
       gni_per_capita_threshold, pricing_multiplier, created_at, updated_at
FROM pricing_zones 
WHERE iso_code = $1;

-- name: GetPricingZoneByCountry :one
SELECT id, country, iso_code, zone, zone_name, world_bank_classification, 
       gni_per_capita_threshold, pricing_multiplier, created_at, updated_at
FROM pricing_zones 
WHERE LOWER(country) = LOWER($1);

-- name: GetPricingZonesByZone :many
SELECT id, country, iso_code, zone, zone_name, world_bank_classification, 
       gni_per_capita_threshold, pricing_multiplier, created_at, updated_at
FROM pricing_zones 
WHERE zone = $1
ORDER BY country;

-- name: ListPricingZones :many
SELECT id, country, iso_code, zone, zone_name, world_bank_classification, 
       gni_per_capita_threshold, pricing_multiplier, created_at, updated_at
FROM pricing_zones 
ORDER BY zone, country;

-- name: UpsertPricingZone :one
INSERT INTO pricing_zones (
    country, iso_code, zone, zone_name, world_bank_classification, 
    gni_per_capita_threshold, pricing_multiplier
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (iso_code) DO UPDATE SET
    country = EXCLUDED.country,
    zone = EXCLUDED.zone,
    zone_name = EXCLUDED.zone_name,
    world_bank_classification = EXCLUDED.world_bank_classification,
    gni_per_capita_threshold = EXCLUDED.gni_per_capita_threshold,
    pricing_multiplier = EXCLUDED.pricing_multiplier,
    updated_at = NOW()
RETURNING id, country, iso_code, zone, zone_name, world_bank_classification, 
          gni_per_capita_threshold, pricing_multiplier, created_at, updated_at;

-- name: DeletePricingZone :exec
DELETE FROM pricing_zones WHERE iso_code = $1;

-- name: CountPricingZones :one
SELECT COUNT(*) FROM pricing_zones;

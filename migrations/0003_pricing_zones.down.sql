-- Drop pricing_zones table and related objects
DROP TRIGGER IF EXISTS trigger_update_pricing_zones_updated_at ON pricing_zones;
DROP FUNCTION IF EXISTS update_pricing_zones_updated_at();
DROP TABLE IF EXISTS pricing_zones;

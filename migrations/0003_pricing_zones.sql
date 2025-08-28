-- Create pricing_zones table for dynamic pricing
CREATE TABLE IF NOT EXISTS pricing_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    country VARCHAR(255) NOT NULL,
    iso_code VARCHAR(2) NOT NULL UNIQUE,
    zone VARCHAR(1) NOT NULL CHECK (zone IN ('A', 'B', 'C', 'D')),
    zone_name VARCHAR(50) NOT NULL,
    world_bank_classification VARCHAR(100),
    gni_per_capita_threshold VARCHAR(50),
    pricing_multiplier DECIMAL(5,2) NOT NULL CHECK (pricing_multiplier >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_pricing_zones_iso_code ON pricing_zones(iso_code);
CREATE INDEX IF NOT EXISTS idx_pricing_zones_country ON pricing_zones(country);
CREATE INDEX IF NOT EXISTS idx_pricing_zones_zone ON pricing_zones(zone);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_pricing_zones_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_pricing_zones_updated_at
    BEFORE UPDATE ON pricing_zones
    FOR EACH ROW
    EXECUTE FUNCTION update_pricing_zones_updated_at();

-- Insert default pricing zones data
INSERT INTO pricing_zones (country, iso_code, zone, zone_name, world_bank_classification, gni_per_capita_threshold, pricing_multiplier) VALUES
-- Zone A (Premium) - High-income countries
('United States', 'US', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Canada', 'CA', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('United Kingdom', 'GB', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Germany', 'DE', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('France', 'FR', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Japan', 'JP', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Australia', 'AU', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Netherlands', 'NL', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Sweden', 'SE', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Switzerland', 'CH', 'A', 'Premium', 'High-income', '>$13935', 1.00),

-- Zone B (Mid-High) - Upper-middle-income countries
('China', 'CN', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Brazil', 'BR', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Russia', 'RU', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Mexico', 'MX', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Turkey', 'TR', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('South Africa', 'ZA', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Malaysia', 'MY', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Thailand', 'TH', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),

-- Zone C (Mid-Low) - Lower-middle-income countries
('India', 'IN', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Indonesia', 'ID', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Philippines', 'PH', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Vietnam', 'VN', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Egypt', 'EG', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Nigeria', 'NG', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Pakistan', 'PK', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Bangladesh', 'BD', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),

-- Zone D (Low-Income) - Low-income countries
('Afghanistan', 'AF', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Ethiopia', 'ET', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Uganda', 'UG', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Tanzania', 'TZ', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Kenya', 'KE', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Mozambique', 'MZ', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Madagascar', 'MG', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Malawi', 'MW', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20)
ON CONFLICT (iso_code) DO UPDATE SET
    country = EXCLUDED.country,
    zone = EXCLUDED.zone,
    zone_name = EXCLUDED.zone_name,
    world_bank_classification = EXCLUDED.world_bank_classification,
    gni_per_capita_threshold = EXCLUDED.gni_per_capita_threshold,
    pricing_multiplier = EXCLUDED.pricing_multiplier,
    updated_at = NOW();

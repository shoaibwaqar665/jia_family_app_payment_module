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

-- Insert pricing zones data from CSV
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
('Norway', 'NO', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Denmark', 'DK', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Finland', 'FI', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Austria', 'AT', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Belgium', 'BE', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Ireland', 'IE', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('New Zealand', 'NZ', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Singapore', 'SG', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Hong Kong', 'HK', 'A', 'Premium', 'High-income', '>$13935', 1.00),
('Israel', 'IL', 'A', 'Premium', 'High-income', '>$13935', 1.00),

-- Zone B (Mid-High) - Upper-middle-income countries
('China', 'CN', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Brazil', 'BR', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Russia', 'RU', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Mexico', 'MX', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Turkey', 'TR', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('South Africa', 'ZA', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Malaysia', 'MY', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Thailand', 'TH', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Argentina', 'AR', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Chile', 'CL', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Colombia', 'CO', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Peru', 'PE', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Romania', 'RO', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Bulgaria', 'BG', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),
('Croatia', 'HR', 'B', 'Mid-High', 'Upper-middle-income', '$4496-$13935', 0.70),

-- Zone C (Mid-Low) - Lower-middle-income countries
('India', 'IN', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Indonesia', 'ID', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Philippines', 'PH', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Vietnam', 'VN', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Egypt', 'EG', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Nigeria', 'NG', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Pakistan', 'PK', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Bangladesh', 'BD', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Ukraine', 'UA', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Morocco', 'MA', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Algeria', 'DZ', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Iraq', 'IQ', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Ghana', 'GH', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Kenya', 'KE', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),
('Tanzania', 'TZ', 'C', 'Mid-Low', 'Lower-middle-income', '$1136-$4495', 0.40),

-- Zone D (Low-Income) - Low-income countries
('Afghanistan', 'AF', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Ethiopia', 'ET', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Uganda', 'UG', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Mozambique', 'MZ', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Madagascar', 'MG', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Malawi', 'MW', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Burkina Faso', 'BF', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Mali', 'ML', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Niger', 'NE', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Chad', 'TD', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Somalia', 'SO', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Central African Republic', 'CF', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Democratic Republic of Congo', 'CD', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Burundi', 'BI', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20),
('Rwanda', 'RW', 'D', 'Low-Income', 'Low-income', '≤$1135', 0.20)
ON CONFLICT (iso_code) DO UPDATE SET
    country = EXCLUDED.country,
    zone = EXCLUDED.zone,
    zone_name = EXCLUDED.zone_name,
    world_bank_classification = EXCLUDED.world_bank_classification,
    gni_per_capita_threshold = EXCLUDED.gni_per_capita_threshold,
    pricing_multiplier = EXCLUDED.pricing_multiplier,
    updated_at = NOW();

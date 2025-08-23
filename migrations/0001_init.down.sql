-- Migration: 0001_init (DOWN)
-- Description: Drop initial plans and entitlements tables structure

-- Drop triggers
DROP TRIGGER IF EXISTS update_entitlements_updated_at ON entitlements;
DROP TRIGGER IF EXISTS update_plans_updated_at ON plans;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (order matters due to foreign key constraints)
DROP TABLE IF EXISTS entitlements;
DROP TABLE IF EXISTS plans;

-- Note: We don't drop extensions as they might be used by other parts of the system
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";

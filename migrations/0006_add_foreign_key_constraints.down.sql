-- Migration: 0006_add_foreign_key_constraints (DOWN)
-- Description: Remove foreign key constraints and additional constraints

-- Remove foreign key constraints
ALTER TABLE entitlements 
DROP CONSTRAINT IF EXISTS fk_entitlements_plan_id;

-- Remove check constraints
ALTER TABLE payments 
DROP CONSTRAINT IF EXISTS check_payments_amount_positive;

ALTER TABLE plans 
DROP CONSTRAINT IF EXISTS check_plans_price_positive;

ALTER TABLE plans 
DROP CONSTRAINT IF EXISTS check_plans_max_users_positive;

ALTER TABLE entitlements 
DROP CONSTRAINT IF EXISTS check_entitlements_status_valid;

ALTER TABLE plans 
DROP CONSTRAINT IF EXISTS check_plans_billing_cycle_valid;

-- Remove unique constraints
ALTER TABLE payments 
DROP CONSTRAINT IF EXISTS unique_payments_stripe_payment_intent_id;

ALTER TABLE payments 
DROP CONSTRAINT IF EXISTS unique_payments_stripe_session_id;

-- Remove partial indexes
DROP INDEX IF EXISTS idx_entitlements_active;
DROP INDEX IF EXISTS idx_payments_pending;
DROP INDEX IF EXISTS idx_plans_active;

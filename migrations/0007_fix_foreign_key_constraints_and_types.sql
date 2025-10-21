-- Migration: 0007_fix_foreign_key_constraints_and_types
-- Description: Fix foreign key constraints and data type mismatches

-- First, let's check if we need to fix the plan_id data type mismatch
-- The domain model uses uuid.UUID but the database uses VARCHAR(100)
-- We'll keep VARCHAR(100) for now as it allows for more flexible plan IDs

-- Add any missing foreign key constraints
-- Note: Payments table doesn't have foreign keys as customer_id is a string identifier
-- and doesn't reference a specific table in this schema

-- Ensure entitlements.plan_id references plans.id (this should already exist)
-- Let's verify and recreate if needed
DO $$
BEGIN
    -- Drop existing constraint if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_entitlements_plan_id' 
        AND table_name = 'entitlements'
    ) THEN
        ALTER TABLE entitlements DROP CONSTRAINT fk_entitlements_plan_id;
    END IF;
    
    -- Add the constraint back with proper settings
    ALTER TABLE entitlements 
    ADD CONSTRAINT fk_entitlements_plan_id 
    FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE ON UPDATE CASCADE;
END $$;

-- Add additional constraints for data integrity

-- Ensure customer_id is not empty in payments
ALTER TABLE payments 
ADD CONSTRAINT check_payments_customer_id_not_empty 
CHECK (LENGTH(TRIM(customer_id)) > 0);

-- Ensure order_id is not empty in payments
ALTER TABLE payments 
ADD CONSTRAINT check_payments_order_id_not_empty 
CHECK (LENGTH(TRIM(order_id)) > 0);

-- Ensure user_id is not empty in entitlements
ALTER TABLE entitlements 
ADD CONSTRAINT check_entitlements_user_id_not_empty 
CHECK (LENGTH(TRIM(user_id)) > 0);

-- Ensure feature_code is not empty in entitlements
ALTER TABLE entitlements 
ADD CONSTRAINT check_entitlements_feature_code_not_empty 
CHECK (LENGTH(TRIM(feature_code)) > 0);

-- Ensure plan_id is not empty in entitlements
ALTER TABLE entitlements 
ADD CONSTRAINT check_entitlements_plan_id_not_empty 
CHECK (LENGTH(TRIM(plan_id)) > 0);

-- Add unique constraints where appropriate

-- Ensure unique order_id per customer (business rule)
ALTER TABLE payments 
ADD CONSTRAINT unique_payments_customer_order 
UNIQUE (customer_id, order_id);

-- Add comments for better documentation
COMMENT ON CONSTRAINT fk_entitlements_plan_id ON entitlements IS 'Foreign key constraint ensuring entitlements reference valid plans';
COMMENT ON CONSTRAINT check_payments_amount_positive ON payments IS 'Ensures payment amounts are positive';
COMMENT ON CONSTRAINT check_payments_customer_id_not_empty ON payments IS 'Ensures customer ID is not empty';
COMMENT ON CONSTRAINT check_payments_order_id_not_empty ON payments IS 'Ensures order ID is not empty';
COMMENT ON CONSTRAINT unique_payments_customer_order ON payments IS 'Ensures unique order ID per customer';
COMMENT ON CONSTRAINT check_entitlements_user_id_not_empty ON entitlements IS 'Ensures user ID is not empty';
COMMENT ON CONSTRAINT check_entitlements_feature_code_not_empty ON entitlements IS 'Ensures feature code is not empty';
COMMENT ON CONSTRAINT check_entitlements_plan_id_not_empty ON entitlements IS 'Ensures plan ID is not empty';

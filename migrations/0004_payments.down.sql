-- Migration: 0004_payments_down
-- Description: Remove payments table

-- Drop trigger first
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;

-- Drop indexes
DROP INDEX IF EXISTS idx_payments_customer_id;
DROP INDEX IF EXISTS idx_payments_order_id;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_payments_external_payment_id;
DROP INDEX IF EXISTS idx_payments_created_at;

-- Drop table
DROP TABLE IF EXISTS payments;

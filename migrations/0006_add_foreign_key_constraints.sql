-- Migration: 0006_add_foreign_key_constraints
-- Description: Add foreign key constraints to improve data integrity

-- Add foreign key constraint for entitlements.plan_id -> plans.id
ALTER TABLE entitlements 
ADD CONSTRAINT fk_entitlements_plan_id 
FOREIGN KEY (plan_id) REFERENCES plans(id) ON DELETE CASCADE;

-- Add foreign key constraint for outbox events (if needed)
-- Note: Outbox events typically don't have foreign keys as they're independent events

-- Add foreign key constraint for webhook events (if they reference other tables)
-- Note: Webhook events are typically independent and don't need foreign keys

-- Add check constraints for better data validation

-- Ensure positive amounts for payments
ALTER TABLE payments 
ADD CONSTRAINT check_payments_amount_positive 
CHECK (amount > 0);

-- Ensure positive prices for plans
ALTER TABLE plans 
ADD CONSTRAINT check_plans_price_positive 
CHECK (price_cents > 0);

-- Ensure max_users is positive for plans
ALTER TABLE plans 
ADD CONSTRAINT check_plans_max_users_positive 
CHECK (max_users > 0);

-- Ensure valid entitlement status
ALTER TABLE entitlements 
ADD CONSTRAINT check_entitlements_status_valid 
CHECK (status IN ('active', 'inactive', 'expired', 'cancelled'));

-- Ensure valid billing cycle
ALTER TABLE plans 
ADD CONSTRAINT check_plans_billing_cycle_valid 
CHECK (billing_cycle IN ('monthly', 'yearly', 'lifetime'));

-- Add unique constraints where appropriate

-- Ensure unique stripe payment intent IDs
ALTER TABLE payments 
ADD CONSTRAINT unique_payments_stripe_payment_intent_id 
UNIQUE (stripe_payment_intent_id);

-- Ensure unique stripe session IDs
ALTER TABLE payments 
ADD CONSTRAINT unique_payments_stripe_session_id 
UNIQUE (stripe_session_id);

-- Add partial indexes for better performance

-- Index for active entitlements only
CREATE INDEX idx_entitlements_active 
ON entitlements(user_id, feature_code) 
WHERE status = 'active';

-- Index for pending payments only
CREATE INDEX idx_payments_pending 
ON payments(customer_id, created_at) 
WHERE status = 'pending';

-- Index for active plans only
CREATE INDEX idx_plans_active 
ON plans(id, name) 
WHERE active = true;

-- Add comments for documentation
COMMENT ON TABLE payments IS 'Stores payment transactions and their status';
COMMENT ON TABLE plans IS 'Stores subscription plans with pricing and features';
COMMENT ON TABLE entitlements IS 'Stores user entitlements to features based on their plans';
COMMENT ON TABLE webhook_events IS 'Stores webhook events for idempotency and audit purposes';
COMMENT ON TABLE outbox IS 'Transactional outbox for reliable event publishing';

COMMENT ON COLUMN payments.amount IS 'Amount in cents (e.g., 1000 = $10.00)';
COMMENT ON COLUMN plans.price_cents IS 'Price in cents (e.g., 999 = $9.99)';
COMMENT ON COLUMN entitlements.expires_at IS 'When the entitlement expires (NULL means never expires)';
COMMENT ON COLUMN webhook_events.processed IS 'Whether the webhook event has been processed';
COMMENT ON COLUMN outbox.status IS 'Status: pending, published, failed';

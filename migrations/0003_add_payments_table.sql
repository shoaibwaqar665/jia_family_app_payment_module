-- Migration: 0003_add_payments_table
-- Description: Add payments table for storing payment records

-- Create payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount INTEGER NOT NULL, -- Amount in cents
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    payment_method VARCHAR(50) NOT NULL,
    customer_id VARCHAR(255) NOT NULL,
    order_id VARCHAR(255) NOT NULL,
    description TEXT,
    stripe_payment_intent_id VARCHAR(255),
    stripe_session_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_payment_status CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled', 'refunded'))
);

-- Create indexes for payments table
CREATE INDEX idx_payments_customer_id ON payments(customer_id);
CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_stripe_payment_intent_id ON payments(stripe_payment_intent_id);
CREATE INDEX idx_payments_stripe_session_id ON payments(stripe_session_id);
CREATE INDEX idx_payments_created_at ON payments(created_at DESC);

-- Create trigger to automatically update updated_at
CREATE TRIGGER update_payments_updated_at 
    BEFORE UPDATE ON payments 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();


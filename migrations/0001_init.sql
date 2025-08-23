-- Migration: 0001_init
-- Description: Create initial plans and entitlements tables structure

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create plans table
CREATE TABLE plans (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    feature_codes TEXT[] NOT NULL,
    billing_cycle VARCHAR(50), -- 'monthly', 'yearly', 'one_time'
    price_cents INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    max_users INTEGER, -- For family plans
    usage_limits JSONB, -- Default limits for features
    metadata JSONB, -- Additional metadata
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create entitlements table
CREATE TABLE entitlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL, -- Spiff ID
    family_id VARCHAR(255), -- NULL for individual plans
    feature_code VARCHAR(100) NOT NULL,
    plan_id VARCHAR(100) NOT NULL,
    subscription_id VARCHAR(255), -- External subscription ID
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP, -- NULL for lifetime purchases
    usage_limits JSONB, -- Feature-specific usage limits
    metadata JSONB, -- Additional feature metadata
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraint
    CONSTRAINT fk_entitlements_plan_id FOREIGN KEY (plan_id) 
        REFERENCES plans(id) ON UPDATE CASCADE
);

-- Create indexes for better query performance

-- Plans table indexes
CREATE INDEX idx_plans_active ON plans(active);
CREATE INDEX idx_plans_billing_cycle ON plans(billing_cycle);
CREATE INDEX idx_plans_feature_codes ON plans USING GIN(feature_codes);

-- Entitlements table indexes
CREATE INDEX idx_entitlements_user_feature ON entitlements(user_id, feature_code);
CREATE INDEX idx_entitlements_family_feature ON entitlements(family_id, feature_code);
CREATE INDEX idx_entitlements_expires_at ON entitlements(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_entitlements_status ON entitlements(status);
CREATE INDEX idx_entitlements_plan_id ON entitlements(plan_id);
CREATE INDEX idx_entitlements_subscription_id ON entitlements(subscription_id);

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to automatically update updated_at
CREATE TRIGGER update_plans_updated_at 
    BEFORE UPDATE ON plans 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_entitlements_updated_at 
    BEFORE UPDATE ON entitlements 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

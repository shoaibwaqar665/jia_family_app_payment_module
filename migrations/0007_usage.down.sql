-- Migration: Add usage tracking table
-- Description: Creates usage table for tracking resource usage and quota management

CREATE TABLE IF NOT EXISTS usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    family_id VARCHAR(255), -- NULL for individual usage
    feature_code VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255) NOT NULL,
    resource_size BIGINT NOT NULL,
    operation VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_usage_user_id ON usage(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_family_id ON usage(family_id);
CREATE INDEX IF NOT EXISTS idx_usage_feature_code ON usage(feature_code);
CREATE INDEX IF NOT EXISTS idx_usage_resource_type ON usage(resource_type);
CREATE INDEX IF NOT EXISTS idx_usage_created_at ON usage(created_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_usage_user_feature ON usage(user_id, feature_code);
CREATE INDEX IF NOT EXISTS idx_usage_user_feature_resource ON usage(user_id, feature_code, resource_type);
CREATE INDEX IF NOT EXISTS idx_usage_user_feature_resource_time ON usage(user_id, feature_code, resource_type, created_at);

-- Add comments for documentation
COMMENT ON TABLE usage IS 'Tracks resource usage for quota management';
COMMENT ON COLUMN usage.resource_type IS 'Type of resource being used (e.g., storage, api_calls, bandwidth)';
COMMENT ON COLUMN usage.resource_size IS 'Size/amount of resource used';
COMMENT ON COLUMN usage.operation IS 'Operation that consumed the resource';
COMMENT ON COLUMN usage.metadata IS 'Additional metadata about the usage';

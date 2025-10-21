-- Migration: 0004_add_outbox_table
-- Description: Add transactional outbox table for reliable event publishing

-- Create outbox table for transactional event publishing
CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP,
    error_message TEXT,
    
    -- Constraints
    CONSTRAINT valid_outbox_status CHECK (status IN ('pending', 'published', 'failed'))
);

-- Create indexes for outbox table
CREATE INDEX idx_outbox_status ON outbox(status);
CREATE INDEX idx_outbox_created_at ON outbox(created_at);
CREATE INDEX idx_outbox_pending ON outbox(status, created_at) WHERE status = 'pending';


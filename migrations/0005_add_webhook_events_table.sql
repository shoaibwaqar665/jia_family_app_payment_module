-- Migration: 0005_add_webhook_events_table
-- Description: Add webhook_events table for idempotency

-- Create webhook_events table for idempotency
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL UNIQUE, -- Stripe event ID
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    signature VARCHAR(255) NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT unique_webhook_event_id UNIQUE (event_id)
);

-- Create indexes for webhook_events table
CREATE INDEX idx_webhook_events_event_id ON webhook_events(event_id);
CREATE INDEX idx_webhook_events_event_type ON webhook_events(event_type);
CREATE INDEX idx_webhook_events_processed ON webhook_events(processed);
CREATE INDEX idx_webhook_events_created_at ON webhook_events(created_at DESC);


-- Migration: 0005_update_amount_to_decimal (DOWN)
-- Description: Revert amount column back to INTEGER for cents

-- First, add a temporary column for the converted cent values
ALTER TABLE payments ADD COLUMN amount_cents INTEGER;

-- Convert existing dollar values to cents (multiply by 100)
UPDATE payments SET amount_cents = (amount * 100)::INTEGER;

-- Drop the old amount column
ALTER TABLE payments DROP COLUMN amount;

-- Rename the new column to amount
ALTER TABLE payments RENAME COLUMN amount_cents TO amount;

-- Make the amount column NOT NULL
ALTER TABLE payments ALTER COLUMN amount SET NOT NULL;

-- Update the comment to reflect the change
COMMENT ON COLUMN payments.amount IS 'Amount in cents (e.g., 1999)';

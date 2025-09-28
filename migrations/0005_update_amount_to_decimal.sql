-- Migration: 0005_update_amount_to_decimal
-- Description: Update amount column to use DECIMAL for dollar amounts instead of INTEGER for cents

-- First, add a temporary column for the converted values
ALTER TABLE payments ADD COLUMN amount_dollars DECIMAL(10,2);

-- Convert existing cent values to dollars (divide by 100)
UPDATE payments SET amount_dollars = amount::DECIMAL / 100.0;

-- Drop the old amount column
ALTER TABLE payments DROP COLUMN amount;

-- Rename the new column to amount
ALTER TABLE payments RENAME COLUMN amount_dollars TO amount;

-- Make the amount column NOT NULL
ALTER TABLE payments ALTER COLUMN amount SET NOT NULL;

-- Update the comment to reflect the change
COMMENT ON COLUMN payments.amount IS 'Amount in dollars (e.g., 19.99)';

-- +goose Up
ALTER TABLE incomes ADD COLUMN IF NOT EXISTS frequency VARCHAR(20) DEFAULT 'monthly';

-- +goose Down
-- Remove the column if it exists (Note: This might fail if the column has data, in which case manual handling is required)
ALTER TABLE incomes DROP COLUMN IF EXISTS frequency;

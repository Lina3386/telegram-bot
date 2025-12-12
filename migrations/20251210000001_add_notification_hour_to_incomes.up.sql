-- +goose Up
ALTER TABLE incomes ADD COLUMN IF NOT EXISTS notification_hour INTEGER DEFAULT 18;

-- +goose Down
ALTER TABLE incomes DROP COLUMN IF EXISTS notification_hour;

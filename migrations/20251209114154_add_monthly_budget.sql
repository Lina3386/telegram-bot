-- +goose Up
ALTER TABLE savings_goals
    ADD COLUMN IF NOT EXISTS monthly_budget_limit BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS monthly_accumulated BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS month_started DATE;

-- +goose Down
ALTER TABLE savings_goals
DROP COLUMN IF EXISTS monthly_budget_limit,
DROP COLUMN IF EXISTS monthly_accumulated,
DROP COLUMN IF EXISTS month_started;

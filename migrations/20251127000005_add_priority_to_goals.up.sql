-- +goose Up
ALTER TABLE savings_goals ADD COLUMN IF NOT EXISTS priority INT DEFAULT 2 CHECK (priority BETWEEN 1 AND 3);

CREATE INDEX IF NOT EXISTS idx_goals_priority ON savings_goals(priority);

-- +goose Down
DROP INDEX IF EXISTS idx_goals_priority;
ALTER TABLE savings_goals DROP COLUMN IF EXISTS priority;


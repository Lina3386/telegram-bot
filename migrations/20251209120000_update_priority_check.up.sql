-- +goose Up
ALTER TABLE savings_goals DROP CONSTRAINT IF EXISTS savings_goals_priority_check;
ALTER TABLE savings_goals ADD CONSTRAINT savings_goals_priority_check CHECK (priority >= 1);

-- +goose Down
ALTER TABLE savings_goals DROP CONSTRAINT IF EXISTS savings_goals_priority_check;
ALTER TABLE savings_goals ADD CONSTRAINT savings_goals_priority_check CHECK (priority BETWEEN 1 AND 3);

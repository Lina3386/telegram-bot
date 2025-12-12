-- +goose Up
CREATE TABLE monthly_contributions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_id BIGINT NOT NULL REFERENCES savings_goals(id) ON DELETE CASCADE,
    month DATE NOT NULL, -- Первый день месяца (например, 2025-12-01)
    amount_contributed BIGINT NOT NULL DEFAULT 0, -- Сумма, внесенная в этом месяце на эту цель
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, goal_id, month)
);

CREATE INDEX idx_monthly_contributions_user_goal_month ON monthly_contributions(user_id, goal_id, month);
CREATE INDEX idx_monthly_contributions_month ON monthly_contributions(month);

-- +goose Down
DROP INDEX IF EXISTS idx_monthly_contributions_month;
DROP INDEX IF EXISTS idx_monthly_contributions_user_goal_month;
DROP TABLE IF EXISTS monthly_contributions CASCADE;

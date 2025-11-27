-- +goose Up
CREATE TABLE IF NOT EXISTS savings_goals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_name VARCHAR(255) NOT NULL,
    target_amount BIGINT NOT NULL,
    current_amount BIGINT DEFAULT 0,
    monthly_contrib BIGINT NOT NULL,
    target_date TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'paused')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    );

CREATE INDEX IF NOT EXISTS idx_goals_user_id ON savings_goals(user_id);
CREATE INDEX IF NOT EXISTS idx_goals_status ON savings_golas(status);
COMMENT ON TABLE savings_goals IS 'Таблица целей накопления';
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_goals_status;
DROP INDEX IF EXISTS idx_goals_user_id;
DROP TABLE IF EXISTS savings_goals CASCADE;
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Up
CREATE TABLE IF NOT EXISTS incomes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    recurring_day INT NOT NULL CHECK (reccuring_day BETWEEN 1 AND 31),
    frequency VARCHAR(50) DEFAULT 'monthly',
    next_pay_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
);

CREATE INDEX IF NOT EXISTS idx_incomes_user_id ON incomes(user_id);
CREATE INDEX IF NOT EXISTS idx_incomes_next_pay_date ON incomes(next_pay_date);

COMMENT ON TABLE incomes IS 'Таблица источников доходов пользователей';
-- +goose StatementBegin
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS idx_incomes_next_pay_date;
DROP INDEX IF EXISTS idx_incomes_user_id;
DROP TABLE IF EXISTS incomes CASCADE;
-- +goose StatementBegin
-- +goose StatementEnd

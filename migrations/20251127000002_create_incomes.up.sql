-- +goose Up
CREATE TABLE IF NOT EXISTS incomes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    frequency VARCHAR(20) DEFAULT 'monthly',
    recurring_day INT NOT NULL,
    next_pay_date TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_incomes_user_id ON incomes(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_incomes_user_id;
DROP TABLE IF EXISTS incomes CASCADE;

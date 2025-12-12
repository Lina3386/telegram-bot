-- +goose Up
-- Initial schema for telegram-bot database

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255) NOT NULL,
    auth_token TEXT,
    monthly_expense BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Incomes table
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

ALTER TABLE incomes ADD COLUMN IF NOT EXISTS frequency VARCHAR(20) DEFAULT 'monthly';

CREATE INDEX IF NOT EXISTS idx_incomes_user_id ON incomes(user_id);

-- Expenses table
CREATE TABLE IF NOT EXISTS expenses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_expenses_user_id ON expenses(user_id);

-- Savings goals table
CREATE TABLE IF NOT EXISTS savings_goals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    goal_name VARCHAR(255) NOT NULL,
    target_amount BIGINT NOT NULL,
    current_amount BIGINT DEFAULT 0,
    monthly_contrib BIGINT DEFAULT 0,
    target_date TIMESTAMP,
    priority INT NOT NULL DEFAULT 2,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_savings_goals_user_id ON savings_goals(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_savings_goals_user_id;
DROP TABLE IF EXISTS savings_goals CASCADE;

DROP INDEX IF EXISTS idx_expenses_user_id;
DROP TABLE IF EXISTS expenses CASCADE;

DROP INDEX IF EXISTS idx_incomes_user_id;
-- Remove the column if it exists (Note: This might fail if the column has data, in which case manual handling is required)
ALTER TABLE incomes DROP COLUMN IF EXISTS frequency;
DROP TABLE IF EXISTS incomes CASCADE;

DROP TABLE IF EXISTS users CASCADE;

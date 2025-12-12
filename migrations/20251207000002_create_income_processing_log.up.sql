-- +goose Up
CREATE TABLE income_processing_log (
    id BIGSERIAL PRIMARY KEY,
    income_id BIGINT NOT NULL REFERENCES incomes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    processed_date DATE NOT NULL, -- Дата, когда был обработан этот доход
    income_amount BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_income_processing_log_income_date ON income_processing_log(income_id, processed_date);
CREATE INDEX idx_income_processing_log_user_date ON income_processing_log(user_id, processed_date);

-- +goose Down
DROP INDEX IF EXISTS idx_income_processing_log_user_date;
DROP INDEX IF EXISTS idx_income_processing_log_income_date;
DROP TABLE IF EXISTS income_processing_log CASCADE;

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"github.com/Lina3386/telegram-bot/internal/models"
)

type IncomeProcessingLogRepository struct {
	db *sql.DB
}

func NewIncomeProcessingLogRepository(db *sql.DB) *IncomeProcessingLogRepository {
	return &IncomeProcessingLogRepository{db: db}
}

func (r *IncomeProcessingLogRepository) CreateProcessingLog(ctx context.Context, incomeID, userID int64, processedDate time.Time, incomeAmount int64) (*models.IncomeProcessingLog, error) {
	log := &models.IncomeProcessingLog{}
	query := `INSERT INTO income_processing_log (income_id, user_id, processed_date, income_amount) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, query, incomeID, userID, processedDate, incomeAmount).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create processing log: %w", err)
	}
	log.IncomeID = incomeID
	log.UserID = userID
	log.ProcessedDate = processedDate
	log.IncomeAmount = incomeAmount
	return log, nil
}

func (r *IncomeProcessingLogRepository) GetProcessingLogByIncomeDate(ctx context.Context, incomeID int64, processedDate time.Time) (*models.IncomeProcessingLog, error) {
	log := &models.IncomeProcessingLog{}
	query := `SELECT id, income_id, user_id, processed_date, income_amount, created_at FROM income_processing_log WHERE income_id = $1 AND processed_date = $2`
	err := r.db.QueryRowContext(ctx, query, incomeID, processedDate).Scan(&log.ID, &log.IncomeID, &log.UserID, &log.ProcessedDate, &log.IncomeAmount, &log.CreatedAt)
	if err != nil {
		return nil, err
	}
	return log, nil
}

func (r *IncomeProcessingLogRepository) GetProcessingLogsByUserDate(ctx context.Context, userID int64, processedDate time.Time) ([]models.IncomeProcessingLog, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, income_id, user_id, processed_date, income_amount, created_at FROM income_processing_log WHERE user_id = $1 AND processed_date = $2`, userID, processedDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.IncomeProcessingLog
	for rows.Next() {
		log := models.IncomeProcessingLog{}
		err := rows.Scan(&log.ID, &log.IncomeID, &log.UserID, &log.ProcessedDate, &log.IncomeAmount, &log.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

func (r *IncomeProcessingLogRepository) IsIncomeProcessedOnDate(ctx context.Context, incomeID int64, processedDate time.Time) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM income_processing_log WHERE income_id = $1 AND processed_date = $2`
	err := r.db.QueryRowContext(ctx, query, incomeID, processedDate).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

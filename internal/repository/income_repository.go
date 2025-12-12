package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
)

type IncomeRepository struct {
	db *sql.DB
}

func NewIncomeRepository(db *sql.DB) *IncomeRepository {
	return &IncomeRepository{db: db}
}

func (r *IncomeRepository) CreateIncome(ctx context.Context, userID int64, name string, amount int64, recurringDay int, nextPayDate time.Time) (*models.Income, error) {
	income := &models.Income{}
	query := `INSERT INTO incomes (user_id, name, amount, recurring_day, next_pay_date) 
	         VALUES ($1, $2, $3, $4, $5) 
	         RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, userID, name, amount, recurringDay, nextPayDate).Scan(&income.ID, &income.CreatedAt, &income.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create income: %w", err)
	}

	income.UserID = userID
	income.Name = name
	income.Amount = amount
	income.RecurringDay = recurringDay
	income.NextPayDate = nextPayDate

	return income, nil
}

func (r *IncomeRepository) CreateIncomeWithFrequency(
	ctx context.Context,
	userID int64,
	name string,
	amount int64,
	frequency string,
	recurringDay int,
	notificationHour int,
	nextPayDate time.Time,
) (*models.Income, error) {
	income := &models.Income{}

	query := `INSERT INTO incomes (user_id, name, amount, frequency, recurring_day, notification_hour, next_pay_date)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				RETURNING id, frequency, notification_hour, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, userID, name, amount, frequency, recurringDay, notificationHour, nextPayDate).Scan(&income.ID, &income.Frequency, &income.NotificationHour, &income.CreatedAt, &income.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create income: %w", err)
	}

	income.UserID = userID
	income.Name = name
	income.Amount = amount
	income.Frequency = frequency
	income.RecurringDay = recurringDay
	income.NotificationHour = notificationHour
	income.NextPayDate = nextPayDate

	return income, nil
}

func (r *IncomeRepository) GetIncomeByID(ctx context.Context, incomeID int64) (*models.Income, error) {
	income := &models.Income{}

	query := `SELECT id, user_id, name, amount, frequency, recurring_day, notification_hour, next_pay_date, created_at, updated_at FROM incomes
				WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, incomeID).
		Scan(&income.ID, &income.UserID, &income.Name, &income.Amount, &income.Frequency, &income.RecurringDay,
			&income.NotificationHour, &income.NextPayDate, &income.CreatedAt, &income.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return income, nil
}

func (r *IncomeRepository) GetUserIncomes(ctx context.Context, userID int64) ([]models.Income, error) {
	query := `SELECT id, user_id, name, amount, frequency, recurring_day, notification_hour, next_pay_date, created_at, updated_at
				FROM incomes WHERE user_id = $1 ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var incomes []models.Income
	for rows.Next() {
		income := models.Income{}
		err := rows.Scan(&income.ID, &income.UserID, &income.Name, &income.Amount,
			&income.Frequency, &income.RecurringDay, &income.NotificationHour, &income.NextPayDate,
			&income.CreatedAt, &income.UpdatedAt)

		if err != nil {
			return nil, err
		}
		incomes = append(incomes, income)
	}

	return incomes, rows.Err()
}

func (r *IncomeRepository) GetIncomesByPayDate(ctx context.Context, payDate time.Time) ([]models.Income, error) {
	year, month, day := payDate.Date()
	localLoc := time.FixedZone("MSK", 3*60*60)
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, localLoc).UTC()
	endOfDay := startOfDay.AddDate(0, 0, 1)

	query := `SELECT id, user_id, name, amount, frequency, recurring_day, notification_hour, next_pay_date, created_at, updated_at
	         FROM incomes
	         WHERE next_pay_date >= $1 AND next_pay_date < $2
	         ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(
		ctx, query,
		startOfDay, endOfDay,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incomes []models.Income
	for rows.Next() {
		income := models.Income{}
		err := rows.Scan(&income.ID, &income.UserID, &income.Name, &income.Amount,
			&income.Frequency, &income.RecurringDay, &income.NotificationHour, &income.NextPayDate,
			&income.CreatedAt, &income.UpdatedAt)

		if err != nil {
			return nil, err
		}
		incomes = append(incomes, income)
	}

	return incomes, rows.Err()
}

func (r *IncomeRepository) GetIncomesByPayDateAndHour(ctx context.Context, payDate time.Time, hour int) ([]models.Income, error) {
	year, month, day := payDate.Date()
	localLoc := time.FixedZone("MSK", 3*60*60)

	startOfHour := time.Date(year, month, day, hour, 0, 0, 0, localLoc).UTC()
	endOfHour := time.Date(year, month, day, hour, 59, 59, 999999999, localLoc).UTC()

	query := `SELECT id, user_id, name, amount, frequency, recurring_day, notification_hour, next_pay_date, created_at, updated_at
	         FROM incomes
	         WHERE next_pay_date >= $1 AND next_pay_date <= $2 AND notification_hour = $3
	         ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(
		ctx, query,
		startOfHour, endOfHour, hour,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incomes []models.Income
	for rows.Next() {
		income := models.Income{}
		err := rows.Scan(&income.ID, &income.UserID, &income.Name, &income.Amount,
			&income.Frequency, &income.RecurringDay, &income.NotificationHour, &income.NextPayDate,
			&income.CreatedAt, &income.UpdatedAt)

		if err != nil {
			return nil, err
		}
		incomes = append(incomes, income)
	}

	return incomes, rows.Err()
}

func (r *IncomeRepository) UpdateIncomeNextPayDate(ctx context.Context, incomeID int64, nextPayDate time.Time) error {
	query := `UPDATE incomes SET next_pay_date = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, nextPayDate, incomeID)
	return err
}

func (r *IncomeRepository) DeleteIncome(ctx context.Context, incomeID int64) error {
	query := `DELETE FROM incomes WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, incomeID)
	return err
}

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Lina3386/telegram-bot/internal/models"
)

type IncomeRepository struct {
	db *sql.DB
}

func NewIncomeRepository(db *sql.DB) *IncomeRepository {
	return &IncomeRepository{db: db}
}

func (r *IncomeRepository) GetUserIncomes(ctx context.Context, userID int64) ([]models.Income, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, user_id, name, amount, recurring_day, next_pay_date, created_at, updated_at 
		 FROM incomes 
		 WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incomes []models.Income
	for rows.Next() {
		income := models.Income{}
		err := rows.Scan(
			&income.ID,
			&income.UserID,
			&income.Name,
			&income.Amount,
			&income.RecurringDay,
			&income.NextPayDate,
			&income.CreatedAt,
			&income.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		incomes = append(incomes, income)
	}

	return incomes, rows.Err()
}

func (r *IncomeRepository) CreateIncome(
	ctx context.Context,
	userID int64,
	name string,
	amount int64,
	recurringDay int,
	nextPayDate interface{},
) (*models.Income, error) {
	income := &models.Income{}
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO incomes (user_id, name, amount, recurring_day, next_pay_date) 
		 VALUES ($1, $2, $3, $4, $5) 
		 RETURNING id, created_at, updated_at`,
		userID, name, amount, recurringDay, nextPayDate,
	).Scan(&income.ID, &income.CreatedAt, &income.UpdatedAt)

	if err != nil {
		return nil, err
	}

	income.UserID = userID
	income.Name = name
	income.Amount = amount
	income.RecurringDay = recurringDay

	return income, nil
}

func (r *IncomeRepository) GetIncomeByID(ctx context.Context, incomeID int64) (*models.Income, error) {
	income := &models.Income{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, name, amount, recurring_day, next_pay_date, created_at, updated_at 
		 FROM incomes 
		 WHERE id = $1`,
		incomeID,
	).Scan(
		&income.ID,
		&income.UserID,
		&income.Name,
		&income.Amount,
		&income.RecurringDay,
		&income.NextPayDate,
		&income.CreatedAt,
		&income.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get income: %w", err)
	}

	return income, nil
}

func (r *IncomeRepository) DeleteIncome(ctx context.Context, incomeID int64) error {
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM incomes WHERE id = $1`,
		incomeID,
	)
	return err
}

func (r *IncomeRepository) GetIncomesByPayDate(ctx context.Context, payDate time.Time) ([]models.Income, error) {
	startOfDay := time.Date(payDate.Year(), payDate.Month(), payDate.Day(), 0, 0, 0, 0, payDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT i.id, i.user_id, i.name, i.amount, i.recurring_day, i.next_pay_date, i.created_at, i.updated_at, u.telegram_id
		 FROM incomes i
		 JOIN users u ON i.user_id = u.id
		 WHERE i.next_pay_date >= $1 AND i.next_pay_date < $2`,
		startOfDay, endOfDay,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incomes []models.Income
	for rows.Next() {
		income := models.Income{}
		var telegramID int64
		err := rows.Scan(
			&income.ID,
			&income.UserID,
			&income.Name,
			&income.Amount,
			&income.RecurringDay,
			&income.NextPayDate,
			&income.CreatedAt,
			&income.UpdatedAt,
			&telegramID,
		)
		if err != nil {
			return nil, err
		}
		income.UserID = telegramID
		incomes = append(incomes, income)
	}

	return incomes, rows.Err()
}

func (r *IncomeRepository) UpdateIncomeNextPayDate(ctx context.Context, incomeID int64, nextPayDate time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE incomes SET next_pay_date = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		nextPayDate, incomeID,
	)
	return err
}
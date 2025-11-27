package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Lina3386/telegram-bot/internal/models"
)

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) GetUserExpenses(ctx context.Context, userID int64) ([]models.Expense, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, name, amount, created_at, updated_at FROM expenses WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		expense := models.Expense{}
		err := rows.Scan(&expense.ID, &expense.UserID, &expense.Name, &expense.Amount, &expense.CreatedAt, &expense.UpdatedAt)
		if err != nil {
			return nil, err
		}
		expenses = append(expenses, expense)
	}

	return expenses, rows.Err()
}

func (r *ExpenseRepository) CreateExpense(ctx context.Context, userID int64, name string, amount int64) (*models.Expense, error) {
	expense := &models.Expense{}
	err := r.db.QueryRowContext(ctx, `INSERT INTO expenses (user_id, name, amount) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`, userID, name, amount).Scan(&expense.ID, &expense.CreatedAt, &expense.UpdatedAt)
	if err != nil {
		return nil, err
	}
	expense.UserID = userID
	expense.Name = name
	expense.Amount = amount

	return expense, nil
}

func (r *ExpenseRepository) GetExpenseByID(ctx context.Context, expenseID int64) (*models.Expense, error) {
	expense := &models.Expense{}
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, name, amount, created_at, updated_at 
		 FROM expenses 
		 WHERE id = $1`,
		expenseID,
	).Scan(&expense.ID, &expense.UserID, &expense.Name, &expense.Amount, &expense.CreatedAt, &expense.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get expense: %w", err)
	}

	return expense, nil
}

func (r *ExpenseRepository) DeleteExpense(ctx context.Context, expenseID int64) error {
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM expenses WHERE id = $1`,
		expenseID,
	)
	return err
}

func (r *ExpenseRepository) UpdateExpense(ctx context.Context, expenseID int64, name string, amount int64) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE expenses SET name = $1, amount = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`,
		name, amount, expenseID,
	)
	return err
}

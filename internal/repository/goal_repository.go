package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
)

type GoalRepository struct {
	db *sql.DB
}

func NewGoalRepository(db *sql.DB) *GoalRepository {
	return &GoalRepository{db: db}
}

func (r *GoalRepository) CreateGoal(ctx context.Context, userID int64, goalName string, targetAmount int64, monthlyContrib int64, targetDate time.Time, priority int) (*models.SavingsGoal, error) {
	goal := &models.SavingsGoal{}
	err := r.db.QueryRowContext(ctx, `INSERT INTO savings_goals (user_id, goal_name, target_amount, monthly_contrib, target_date, priority) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`, userID, goalName, targetAmount, monthlyContrib, targetDate, priority).Scan(&goal.ID, &goal.CreatedAt, &goal.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create goal: %w", err)
	}
	goal.UserID = userID
	goal.GoalName = goalName
	goal.TargetAmount = targetAmount
	goal.MonthlyContrib = monthlyContrib
	goal.TargetDate = targetDate
	goal.Priority = priority
	goal.Status = "active"
	return goal, nil
}

func (r *GoalRepository) GetUserActiveGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, goal_name, target_amount, current_amount, monthly_contrib, monthly_budget_limit, monthly_accumulated, month_started, target_date, priority, status, created_at, updated_at FROM savings_goals WHERE user_id = $1 AND status = 'active' ORDER BY priority ASC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.SavingsGoal
	for rows.Next() {
		goal := models.SavingsGoal{}
		err := rows.Scan(&goal.ID, &goal.UserID, &goal.GoalName, &goal.TargetAmount, &goal.CurrentAmount, &goal.MonthlyContrib, &goal.MonthlyBudgetLimit, &goal.MonthlyAccumulated, &goal.MonthStarted, &goal.TargetDate, &goal.Priority, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, rows.Err()
}

func (r *GoalRepository) GetUserGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, goal_name, target_amount, current_amount, monthly_contrib, monthly_budget_limit, monthly_accumulated, month_started, target_date, priority, status, created_at, updated_at FROM savings_goals WHERE user_id = $1 ORDER BY priority ASC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.SavingsGoal
	for rows.Next() {
		goal := models.SavingsGoal{}
		err := rows.Scan(&goal.ID, &goal.UserID, &goal.GoalName, &goal.TargetAmount, &goal.CurrentAmount, &goal.MonthlyContrib, &goal.MonthlyBudgetLimit, &goal.MonthlyAccumulated, &goal.MonthStarted, &goal.TargetDate, &goal.Priority, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, rows.Err()
}

func (r *GoalRepository) UpdateGoal(ctx context.Context, goal *models.SavingsGoal) error {
	query := `
        UPDATE savings_goals
        SET goal_name = $1,
            target_amount = $2,
            current_amount = $3,
            monthly_contrib = $4,
            monthly_budget_limit = $5,
            monthly_accumulated = $6,
            month_started = $7,
            target_date = $8,
            priority = $9,
            status = $10,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $11
    `

	_, err := r.db.ExecContext(ctx, query,
		goal.GoalName,
		goal.TargetAmount,
		goal.CurrentAmount,
		goal.MonthlyContrib,
		goal.MonthlyBudgetLimit,
		goal.MonthlyAccumulated,
		goal.MonthStarted,
		goal.TargetDate,
		goal.Priority,
		goal.Status,
		goal.ID,
	)

	return err
}

func (r *GoalRepository) GetGoalByID(ctx context.Context, goalID int64) (*models.SavingsGoal, error) {
	goal := &models.SavingsGoal{}
	query := `
        SELECT id, user_id, goal_name, target_amount, current_amount,
               monthly_contrib, monthly_budget_limit, monthly_accumulated,
               month_started, target_date, priority, status, created_at, updated_at
        FROM savings_goals
        WHERE id = $1
    `

	err := r.db.QueryRowContext(ctx, query, goalID).Scan(
		&goal.ID, &goal.UserID, &goal.GoalName, &goal.TargetAmount,
		&goal.CurrentAmount, &goal.MonthlyContrib, &goal.MonthlyBudgetLimit,
		&goal.MonthlyAccumulated, &goal.MonthStarted, &goal.TargetDate,
		&goal.Priority, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt,
	)

	return goal, err
}

func (r *GoalRepository) DeleteGoal(ctx context.Context, goalID int64) error {
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM savings_goals WHERE id = $1`,
		goalID,
	)
	return err
}

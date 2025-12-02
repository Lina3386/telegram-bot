package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/models"
	"time"
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

func (r *GoalRepository) GetGoalByID(ctx context.Context, goalID int64) (*models.SavingsGoal, error) {
	goal := &models.SavingsGoal{}
	err := r.db.QueryRowContext(ctx, `SELECT id, user_id, goal_name, target_amount, current_amount, monthly_contrib, target_date, priority, status, created_at, updated_at FROM savings_goals WHERE id = $1`, goalID).Scan(&goal.ID, &goal.UserID, &goal.GoalName, &goal.TargetAmount, &goal.CurrentAmount, &goal.MonthlyContrib, &goal.TargetDate, &goal.Priority, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return goal, nil
}

func (r *GoalRepository) GetUserActiveGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, goal_name, target_amount, current_amount, monthly_contrib, target_date, priority, status, created_at, updated_at FROM savings_goals WHERE user_id = $1 AND status = 'active' ORDER BY priority ASC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.SavingsGoal
	for rows.Next() {
		goal := models.SavingsGoal{}
		err := rows.Scan(&goal.ID, &goal.UserID, &goal.GoalName, &goal.TargetAmount, &goal.CurrentAmount, &goal.MonthlyContrib, &goal.TargetDate, &goal.Priority, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, rows.Err()
}

func (r *GoalRepository) GetUserGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, goal_name, target_amount, current_amount, monthly_contrib, target_date, priority, status, created_at, updated_at FROM savings_goals WHERE user_id = $1 ORDER BY priority ASC, created_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []models.SavingsGoal
	for rows.Next() {
		goal := models.SavingsGoal{}
		err := rows.Scan(&goal.ID, &goal.UserID, &goal.GoalName, &goal.TargetAmount, &goal.CurrentAmount, &goal.MonthlyContrib, &goal.TargetDate, &goal.Priority, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, rows.Err()
}

func (r *GoalRepository) UpdateGoal(ctx context.Context, goal *models.SavingsGoal) error {
	_, err := r.db.ExecContext(ctx, `UPDATE savings_goals SET current_amount = $1, status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $3`, goal.CurrentAmount, goal.Status, goal.ID)
	return err
}

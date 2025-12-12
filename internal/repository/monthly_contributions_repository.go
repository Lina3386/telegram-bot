package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"github.com/Lina3386/telegram-bot/internal/models"
)

type MonthlyContributionsRepository struct {
	db *sql.DB
}

func NewMonthlyContributionsRepository(db *sql.DB) *MonthlyContributionsRepository {
	return &MonthlyContributionsRepository{db: db}
}

func (r *MonthlyContributionsRepository) CreateContribution(ctx context.Context, userID, goalID int64, month time.Time, amountContributed int64) (*models.MonthlyContribution, error) {
	contribution := &models.MonthlyContribution{}
	query := `INSERT INTO monthly_contributions (user_id, goal_id, month, amount_contributed) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, goal_id, month) DO NOTHING RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, userID, goalID, month, amountContributed).Scan(&contribution.ID, &contribution.CreatedAt, &contribution.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create contribution: %w", err)
	}
	contribution.UserID = userID
	contribution.GoalID = goalID
	contribution.Month = month
	contribution.AmountContributed = amountContributed
	return contribution, nil
}

func (r *MonthlyContributionsRepository) GetContributionByUserGoalMonth(ctx context.Context, userID, goalID int64, month time.Time) (*models.MonthlyContribution, error) {
	contribution := &models.MonthlyContribution{}
	query := `SELECT id, user_id, goal_id, month, amount_contributed, created_at, updated_at FROM monthly_contributions WHERE user_id = $1 AND goal_id = $2 AND month = $3`
	err := r.db.QueryRowContext(ctx, query, userID, goalID, month).Scan(&contribution.ID, &contribution.UserID, &contribution.GoalID, &contribution.Month, &contribution.AmountContributed, &contribution.CreatedAt, &contribution.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return contribution, nil
}

func (r *MonthlyContributionsRepository) UpdateContribution(ctx context.Context, contribution *models.MonthlyContribution) error {
	query := `UPDATE monthly_contributions SET amount_contributed = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, contribution.AmountContributed, contribution.ID)
	return err
}

func (r *MonthlyContributionsRepository) GetUserContributionsByMonth(ctx context.Context, userID int64, month time.Time) ([]models.MonthlyContribution, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, goal_id, month, amount_contributed, created_at, updated_at FROM monthly_contributions WHERE user_id = $1 AND month = $2`, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contributions []models.MonthlyContribution
	for rows.Next() {
		contribution := models.MonthlyContribution{}
		err := rows.Scan(&contribution.ID, &contribution.UserID, &contribution.GoalID, &contribution.Month, &contribution.AmountContributed, &contribution.CreatedAt, &contribution.UpdatedAt)
		if err != nil {
			return nil, err
		}
		contributions = append(contributions, contribution)
	}

	return contributions, rows.Err()
}

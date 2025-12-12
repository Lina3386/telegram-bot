package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO users (telegram_id, username, auth_token) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		user.TelegramID, user.Username, user.AuthToken).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, telegram_id, username, auth_token, monthly_expense, created_at, updated_at FROM users WHERE telegram_id = $1`, telegramID,
	).Scan(&user.ID, &user.TelegramID, &user.Username, &user.AuthToken,
		&user.MonthlyExpense, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, telegram_id, username, auth_token, monthly_expense, created_at, updated_at FROM users WHERE id = $1`, userID,
	).Scan(&user.ID, &user.TelegramID, &user.Username, &user.AuthToken,
		&user.MonthlyExpense, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) UpdateMonthlyExpense(ctx context.Context, userID int64, expense int64) error {
	_, err := r.db.ExecContext(
		ctx, `UPDATE users
		SET monthly_expense = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`, expense, userID,
	)
	return err
}

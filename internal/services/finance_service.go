package services

import (
	"context"
	"github.com/Lina3386/telegram-bot/internal/models"
	"time"

	"database/sql"
	"github.com/Lina3386/telegram-bot/internal/repository"
)

type FinanceService struct {
	userRepo    *repository.UserRepository
	incomeRepo  *repository.IncomeRepository
	expenseRepo *repository.ExpenseRepository
	goalRepo    *repository.GoalRepository
}

func NewFinanceService(db *sql.DB) *FinanceService {
	return &FinanceService{
		userRepo:    repository.NewUserRepository(db),
		incomeRepo:  repository.NewIncomeRepository(db),
		expenseRepo: repository.NewExpenseRepository(db),
		goalRepo:    repository.NewGoalRepository(db),
	}
}

// Считает общий доход
func (s *FinanceService) CalculateTotalIncome(ctx context.Context, userID int64) (int64, error) {
	incomes, err := s.incomeRepo.GetUserIncomes(ctx, userID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, income := range incomes {
		total += income.Amount
	}
	return total, nil
}

// Считает сумму всех расходов
func (s *FinanceService) CalculateTotalExpense(ctx context.Context, userID int64) (int64, error) {
	expenses, err := s.expenseRepo.GetUserExpenses(ctx, userID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, expense := range expenses {
		total += expense.Amount
	}
	return total, nil
}

// Рассчитывает доступный бюджет: доходы - расходы
func (s *FinanceService) CalculateAvailableForSavings(ctx context.Context, userID int64) (int64, error) {
	totalIncome, err := s.CalculateTotalIncome(ctx, userID)
	if err != nil {
		return 0, err
	}

	totalExpense, err := s.CalculateTotalExpense(ctx, userID)
	if err != nil {
		return 0, err
	}

	available := totalIncome + totalExpense
	if available < 0 {
		available = 0
	}

	return available, nil
}

// Создаёт цель с расчётом
func (s *FinanceService) CreateGoal(ctx context.Context, userID int64, goalName string, targetAmount int64) (*models.SavingsGoal, error) {
	availableForSavings, err := s.CalculateAvailableForSavings(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Сколько месяцев до достижения цели
	monthsNeeded := (targetAmount + availableForSavings - 1) / availableForSavings
	targetDate := time.Now().AddDate(0, int(monthsNeeded), 0)

	// Создаём цель
	goal, err := s.goalRepo.CreateGoal(
		ctx,
		userID,
		goalName,
		targetAmount,
		availableForSavings,
		targetDate,
	)

	return goal, err
}

// Добавляет деньги к цели
func (s *FinanceService) ContributeToGoal(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	goal.CurrentAmount += amount

	if goal.CurrentAmount >= goal.TargetAmount {
		goal.Status = "completed"
	}

	err = s.goalRepo.UpdateGoal(ctx, goal)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"database/sql"
	"github.com/Lina3386/telegram-bot/internal/models"
	"github.com/Lina3386/telegram-bot/internal/repository"
)

type FinanceService struct {
	userRepo    repository.UserRepository
	incomeRepo  repository.IncomeRepository
	expenseRepo repository.ExpenseRepository
	goalRepo    repository.GoalRepository
}

func NewFinanceService(
	userRepo repository.UserRepository,
	incomeRepo repository.IncomeRepository,
	expenseRepo repository.ExpenseRepository,
	goalRepo repository.GoalRepository,
) *FinanceService {
	return &FinanceService{
		userRepo:    userRepo,
		incomeRepo:  incomeRepo,
		expenseRepo: expenseRepo,
		goalRepo:    goalRepo,
	}
}

func (s *FinanceService) CreateUser(ctx context.Context, telegramID int64, username string, authToken string) (*models.User, error) {
	user, err := s.userRepo.CreateUser(ctx, telegramID, username, authToken)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return nil, err
	}
	log.Printf("User created: ID=%d, TelegramID=%d, Username=%s", user.ID, telegramID, username)
	return user, nil
}

func (s *FinanceService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	return s.userRepo.GetUserByTelegramID(ctx, telegramID)
}

func (s *FinanceService) CreateIncome(
	ctx context.Context,
	telegramID int64,
	name string,
	amount int64,
	recurringDay int,
	nextPayDate time.Time,
) (*models.Income, error) {
	// ✅ Получаем пользователя по telegram_id, чтобы получить внутренний ID
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// ✅ Используем внутренний ID пользователя
	income, err := s.incomeRepo.CreateIncome(ctx, user.ID, name, amount, recurringDay, nextPayDate)
	if err != nil {
		log.Printf("Failed to create income: %v", err)
		return nil, err
	}
	log.Printf("Income created: %s, Amount=%d, RecurringDay=%d", name, amount, recurringDay)
	return income, nil
}

func (s *FinanceService) GetUserIncomes(ctx context.Context, telegramID int64) ([]models.Income, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.incomeRepo.GetUserIncomes(ctx, user.ID)
}

func (s *FinanceService) CalculateTotalIncome(ctx context.Context, telegramID int64) (int64, error) {
	incomes, err := s.GetUserIncomes(ctx, telegramID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, income := range incomes {
		total += income.Amount
	}
	return total, nil
}

func (s *FinanceService) GetIncomesByPayDate(ctx context.Context, payDate time.Time) ([]models.Income, error) {
	return s.incomeRepo.GetIncomesByPayDate(ctx, payDate)
}

func (s *FinanceService) UpdateIncomeNextPayDate(ctx context.Context, incomeID int64, nextPayDate time.Time) error {
	return s.incomeRepo.UpdateIncomeNextPayDate(ctx, incomeID, nextPayDate)
}

func (s *FinanceService) CreateExpense(ctx context.Context, telegramID int64, name string, amount int64) (*models.Expense, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	expense, err := s.expenseRepo.CreateExpense(ctx, user.ID, name, amount)
	if err != nil {
		log.Printf("Failed to create expense: %v", err)
		return nil, err
	}
	log.Printf("Expense created: %s, Amount=%d", name, amount)
	return expense, nil
}


func (s *FinanceService) GetUserExpenses(ctx context.Context, telegramID int64) ([]models.Expense, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.expenseRepo.GetUserExpenses(ctx, user.ID)
}

func (s *FinanceService) CalculateTotalExpense(ctx context.Context, telegramID int64) (int64, error) {
	expenses, err := s.GetUserExpenses(ctx, telegramID)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, expense := range expenses {
		total += expense.Amount
	}
	return total, nil
}


func (s *FinanceService) CreateGoal(ctx context.Context, telegramID int64, goalName string, targetAmount int64, priority int) (*models.SavingsGoal, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	availableForSavings, err := s.CalculateAvailableForSavings(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	// ✅ Получаем активные цели по внутреннему ID пользователя
	activeGoals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	monthlyContrib := s.distributeFundsToGoal(availableForSavings, activeGoals, priority)

	if monthlyContrib == 0 {
		monthlyContrib = 1000 // Минимальный взнос
	}

	// месяцев до достижения цели
	monthsNeeded := (targetAmount + monthlyContrib - 1) / monthlyContrib
	if monthsNeeded == 0 {
		monthsNeeded = 1
	}

	targetDate := time.Now().AddDate(0, int(monthsNeeded), 0)

	// ✅ Используем внутренний ID пользователя
	goal, err := s.goalRepo.CreateGoal(
		ctx,
		user.ID,
		goalName,
		targetAmount,
		monthlyContrib,
		targetDate,
		priority,
	)

	if err != nil {
		log.Printf("Failed to create goal: %v", err)
		return nil, err
	}

	log.Printf("Goal created: %s, Target=%d, MonthlyContrib=%d, Priority=%d, TargetDate=%s",
		goalName, targetAmount, monthlyContrib, priority, targetDate.Format("02.01.2006"))

	return goal, nil
}

// распределение средств между целями по приоритету
func (s *FinanceService) distributeFundsToGoal(availableFunds int64, existingGoals []models.SavingsGoal, newPriority int) int64 {
	if len(existingGoals) == 0 {
		return availableFunds
	}

	totalPriorityWeight := int64(0)
	for _, goal := range existingGoals {
		weight := int64(4 - goal.Priority)
		totalPriorityWeight += weight
	}

	newWeight := int64(4 - newPriority)
	totalPriorityWeight += newWeight

	newGoalShare := (availableFunds * newWeight) / totalPriorityWeight

	return newGoalShare
}

func (s *FinanceService) GetUserGoals(ctx context.Context, telegramID int64) ([]models.SavingsGoal, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.goalRepo.GetUserGoals(ctx, user.ID)
}

func (s *FinanceService) GetUserActiveGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	return s.goalRepo.GetUserActiveGoals(ctx, userID)
}

func (s *FinanceService) GetUserActiveGoalsByTelegramID(ctx context.Context, telegramID int64) ([]models.SavingsGoal, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.goalRepo.GetUserActiveGoals(ctx, user.ID)
}

func (s *FinanceService) GetUserGoalByID(ctx context.Context, telegramID int64, goalID int64) (*models.SavingsGoal, error) {
	// ✅ Получаем пользователя по telegram_id
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	// ✅ Проверяем по внутреннему ID пользователя
	if goal.UserID != user.ID {
		return nil, fmt.Errorf("goal does not belong to user")
	}
	return goal, nil
}

func (s *FinanceService) ContributeToGoal(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	goal.CurrentAmount += amount

	if goal.CurrentAmount >= goal.TargetAmount {
		goal.Status = "completed"
		goal.CurrentAmount = goal.TargetAmount // Не даем превысить целевую сумму
		log.Printf("Goal %d reached!", goalID)
	}

	err = s.goalRepo.UpdateGoal(ctx, goal)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

func (s *FinanceService) WithdrawFromGoal(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	if goal.CurrentAmount < amount {
		goal.CurrentAmount = 0
	} else {
		goal.CurrentAmount -= amount
	}

	if goal.Status == "completed" && goal.CurrentAmount < goal.TargetAmount {
		goal.Status = "active"
	}

	err = s.goalRepo.UpdateGoal(ctx, goal)
	if err != nil {
		return nil, err
	}

	log.Printf("Withdrew %d from goal %d, new amount: %d", amount, goalID, goal.CurrentAmount)
	return goal, nil
}

func (s *FinanceService) CalculateAvailableForSavings(ctx context.Context, telegramID int64) (int64, error) {
	totalIncome, err := s.CalculateTotalIncome(ctx, telegramID)
	if err != nil {
		return 0, err
	}

	totalExpense, err := s.CalculateTotalExpense(ctx, telegramID)
	if err != nil {
		return 0, err
	}

	available := totalIncome - totalExpense
	if available < 0 {
		available = 0
	}

	return available, nil
}

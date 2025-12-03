package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
	"github.com/Lina3386/telegram-bot/internal/repository"
)

type FinanceService struct {
	userRepo    *repository.UserRepository
	incomeRepo  *repository.IncomeRepository
	expenseRepo *repository.ExpenseRepository
	goalRepo    *repository.GoalRepository
}

func NewFinanceService(userRepo *repository.UserRepository, incomeRepo *repository.IncomeRepository, expenseRepo *repository.ExpenseRepository, goalRepo *repository.GoalRepository) *FinanceService {
	return &FinanceService{
		userRepo:    userRepo,
		incomeRepo:  incomeRepo,
		expenseRepo: expenseRepo,
		goalRepo:    goalRepo,
	}
}

// создает нового пользователя
func (s *FinanceService) CreateUser(ctx context.Context, telegramID int64, username string, authToken string) (*models.User, error) {
	user, err := s.userRepo.CreateUser(ctx, telegramID, username, authToken)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		return nil, err
	}
	log.Printf("User created: ID=%d, TelegramID=%d, Username=%s", user.ID, telegramID, username)
	return user, nil
}

// получает пользователя по Telegram ID
func (s *FinanceService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	return s.userRepo.GetUserByTelegramID(ctx, telegramID)
}

// создает новый источник дохода
func (s *FinanceService) CreateIncome(
	ctx context.Context,
	userID int64,
	name string,
	amount int64,
	recurringDay int,
	nextPayDate time.Time,
) (*models.Income, error) {
	income, err := s.incomeRepo.CreateIncome(ctx, userID, name, amount, recurringDay, nextPayDate)
	if err != nil {
		log.Printf("Failed to create income: %v", err)
		return nil, err
	}
	log.Printf("Income created: %s, Amount=%d, RecurringDay=%d", name, amount, recurringDay)
	return income, nil
}

// получает все доходы пользователя
func (s *FinanceService) GetUserIncomes(ctx context.Context, userID int64) ([]models.Income, error) {
	return s.incomeRepo.GetUserIncomes(ctx, userID)
}

// считает общий доход
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

func (s *FinanceService) GetIncomesByPayDate(ctx context.Context, payDate time.Time) ([]models.Income, error) {
	return s.incomeRepo.GetIncomesByPayDate(ctx, payDate)
}

func (s *FinanceService) UpdateIncomeNextPayDate(ctx context.Context, incomeID int64, nextPayDate time.Time) error {
	return s.incomeRepo.UpdateIncomeNextPayDate(ctx, incomeID, nextPayDate)
}

func (s *FinanceService) CreateExpense(ctx context.Context, userID int64, name string, amount int64) (*models.Expense, error) {
	expense, err := s.expenseRepo.CreateExpense(ctx, userID, name, amount)
	if err != nil {
		log.Printf("Failed to create expense: %v", err)
		return nil, err
	}
	log.Printf("Expense created: %s, Amount=%d", name, amount)
	return expense, nil
}

// получает все расходы пользователя
func (s *FinanceService) GetUserExpenses(ctx context.Context, userID int64) ([]models.Expense, error) {
	return s.expenseRepo.GetUserExpenses(ctx, userID)
}

// считает сумму всех расходов
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

func (s *FinanceService) CreateGoal(ctx context.Context, userID int64, goalName string, targetAmount int64, priority int) (*models.SavingsGoal, error) {
	availableForSavings, err := s.CalculateAvailableForSavings(ctx, userID)
	if err != nil {
		return nil, err
	}

	activeGoals, err := s.goalRepo.GetUserActiveGoals(ctx, userID)
	if err != nil {
		return nil, err
	}

	monthlyContrib := s.distributeFundsToGoal(availableForSavings, activeGoals, priority)

	if monthlyContrib == 0 {
		monthlyContrib = 1000
	}

	// месяцев до достижения цели
	monthsNeeded := (targetAmount + monthlyContrib - 1) / monthlyContrib
	if monthsNeeded == 0 {
		monthsNeeded = 1
	}

	targetDate := time.Now().AddDate(0, int(monthsNeeded), 0)

	// Создаем цель
	goal, err := s.goalRepo.CreateGoal(
		ctx,
		userID,
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

func (s *FinanceService) GetUserGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	return s.goalRepo.GetUserGoals(ctx, userID)
}

func (s *FinanceService) GetUserActiveGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	return s.goalRepo.GetUserActiveGoals(ctx, userID)
}

func (s *FinanceService) GetUserActiveGoalsByTelegramID(ctx context.Context, telegramID int64) ([]models.SavingsGoal, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	return s.goalRepo.GetUserActiveGoals(ctx, user.ID)
}

func (s *FinanceService) GetUserGoalByID(ctx context.Context, userID int64, goalID int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	if goal.UserID != userID {
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

func (s *FinanceService) CalculateAvailableForSavings(ctx context.Context, userID int64) (int64, error) {
	totalIncome, err := s.CalculateTotalIncome(ctx, userID)
	if err != nil {
		return 0, err
	}

	totalExpense, err := s.CalculateTotalExpense(ctx, userID)
	if err != nil {
		return 0, err
	}

	available := totalIncome - totalExpense
	if available < 0 {
		available = 0
	}

	return available, nil
}

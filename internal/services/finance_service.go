package services

import (
	"context"
	"log"
	"time"

	"database/sql"
	"github.com/Lina3386/telegram-bot/internal/models"
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

// создает новый расход
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

func (s *FinanceService) CreateGoal(ctx context.Context, userID int64, goalName string, targetAmount int64) (*models.SavingsGoal, error) {
	availableForSavings, err := s.CalculateAvailableForSavings(ctx, userID)
	if err != nil {
		return nil, err
	}

	if availableForSavings == 0 {
		availableForSavings = 1000
	}

	// Сколько месяцев до достижения цели
	monthsNeeded := (targetAmount + availableForSavings - 1) / availableForSavings
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
		availableForSavings,
		targetDate,
	)

	if err != nil {
		log.Printf("Failed to create goal: %v", err)
		return nil, err
	}

	log.Printf("Goal created: %s, Target=%d, MonthlyContrib=%d, TargetDate=%s",
		goalName, targetAmount, availableForSavings, targetDate.Format("02.01.2006"))

	return goal, nil
}

// получает все цели пользователя
func (s *FinanceService) GetUserGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	return s.goalRepo.GetUserGoals(ctx, userID)
}

// GetUserActiveGoals получает активные цели
func (s *FinanceService) GetUserActiveGoals(ctx context.Context, userID int64) ([]models.SavingsGoal, error) {
	return s.goalRepo.GetUserActiveGoals(ctx, userID)
}

// добавляет деньги к цели
func (s *FinanceService) ContributeToGoal(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	goal.CurrentAmount += amount

	if goal.CurrentAmount >= goal.TargetAmount {
		goal.Status = "completed"
		log.Printf("Goal %d reached!", goalID)
	}

	err = s.goalRepo.UpdateGoal(ctx, goal)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

// рассчитывает доступный бюджет: доходы - расходы
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

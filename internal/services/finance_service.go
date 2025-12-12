package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
	"github.com/Lina3386/telegram-bot/internal/repository"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type FinanceService struct {
	incomeRepo         *repository.IncomeRepository
	expenseRepo        *repository.ExpenseRepository
	goalRepo           *repository.GoalRepository
	userRepo           *repository.UserRepository
	processingLogRepo  *repository.IncomeProcessingLogRepository
	monthlyContribRepo *repository.MonthlyContributionsRepository
}

func NewFinanceService(userRepo *repository.UserRepository, incomeRepo *repository.IncomeRepository, expenseRepo *repository.ExpenseRepository, goalRepo *repository.GoalRepository, monthlyContribRepo *repository.MonthlyContributionsRepository, processingLogRepo *repository.IncomeProcessingLogRepository) *FinanceService {
	return &FinanceService{
		userRepo:           userRepo,
		incomeRepo:         incomeRepo,
		expenseRepo:        expenseRepo,
		goalRepo:           goalRepo,
		monthlyContribRepo: monthlyContribRepo,
		processingLogRepo:  processingLogRepo,
	}
}

func (s *FinanceService) CalculateTotalIncome(ctx context.Context, telegramID int64) (int64, error) {
	incomes, err := s.GetUserIncomes(ctx, telegramID)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	year, month, _ := now.Date()

	var total int64
	log.Printf("[INCOME_CALC] Starting calculation for %d-%d", year, month)

	for _, income := range incomes {
		amount := int64(0)

		switch income.Frequency {
		case "monthly":
			amount = income.Amount
			log.Printf("[INCOME_CALC] Monthly '%s': %d₽", income.Name, amount)

		case "weekly":
			weekdayCount := countWeekdaysInMonth(year, month, income.RecurringDay)
			amount = income.Amount * int64(weekdayCount)
			log.Printf("[INCOME_CALC] Weekly '%s': %d₽/week × %d times = %d₽ (day=%d)",
				income.Name, income.Amount, weekdayCount, amount, income.RecurringDay)

		case "biweekly":
			biweeklyCount := countBiweeklyOccurrences(year, month, income.RecurringDay)
			amount = income.Amount * int64(biweeklyCount)

			log.Printf("[INCOME_CALC] Biweekly '%s': %d₽ × %d times = %d₽ (day=%d)",
				income.Name, income.Amount, biweeklyCount, amount, income.RecurringDay)

		default:
			log.Printf("[INCOME_CALC] Unknown frequency '%s' for '%s', skipping", income.Frequency, income.Name)
		}

		total += amount
	}

	log.Printf("[INCOME_CALC] TOTAL INCOME: %d₽", total)
	return total, nil
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func countWeekdaysInMonth(year int, month time.Month, targetWeekday int) int {
	daysInMonth := daysInMonth(year, month)
	count := 0

	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		weekday := int(date.Weekday())
		if weekday == targetWeekday {
			count++
			log.Printf("  [DEBUG] %d-%02d-%02d is weekday %d (target=%d) ✓", year, month, day, weekday, targetWeekday)
		}
	}
	log.Printf("[WEEKDAY_COUNT] Year=%d, Month=%d, TargetWeekday=%d, Count=%d", year, month, targetWeekday, count)

	return count
}

func countBiweeklyOccurrences(year int, month time.Month, targetWeekday int) int {
	daysInMonth := daysInMonth(year, month)
	firstOccurrence := -1

	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		if int(date.Weekday()) == targetWeekday {
			firstOccurrence = day
			break
		}
	}

	if firstOccurrence == -1 {
		log.Printf("[BIWEEKLY_COUNT] Year=%d, Month=%d, TargetWeekday=%d, FirstOccurrence=NOT_FOUND, Count=0", year, month, targetWeekday)
		return 0
	}

	// через 14 дней от первого вхождения
	count := 1
	for day := firstOccurrence + 14; day <= daysInMonth; day += 14 {
		count++
	}

	log.Printf("[BIWEEKLY_COUNT] Year=%d, Month=%d, TargetWeekday=%d, FirstOccurrence=%d, Count=%d",
		year, month, targetWeekday, firstOccurrence, count)
	return count
}

func (s *FinanceService) DistributeFundsToGoals(ctx context.Context, telegramID int64) ([]models.SavingsGoal, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	availableForSavings, err := s.CalculateAvailableForSavings(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	if availableForSavings <= 0 {
		goals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		return goals, nil
	}

	goals, err := s.goalRepo.GetUserActiveGoals(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	if len(goals) == 0 {
		return goals, nil
	}

	for i := 0; i < len(goals)-1; i++ {
		for j := i + 1; j < len(goals); j++ {
			if goals[i].Priority > goals[j].Priority {
				goals[i], goals[j] = goals[j], goals[i]
			}
		}
	}

	n := int64(len(goals))

	summaryFactorial := (n * (n + 1)) / 2

	if summaryFactorial == 0 {
		summaryFactorial = 1
	}

	// пропорционально приоритетам с учетом остатка до цели
	allocated := make([]int64, len(goals))
	totalAllocated := int64(0)

	log.Printf("[DISTRIBUTION] Starting with availableForSavings=%d₽ for %d goals", availableForSavings, n)

	for i := range goals {
		if goals[i].Status == "active" {
			priorityWeight := n - int64(goals[i].Priority) + 1

			contrib := (availableForSavings * priorityWeight) / summaryFactorial

			remainingToTarget := goals[i].TargetAmount - goals[i].CurrentAmount
			if remainingToTarget < 0 {
				remainingToTarget = 0
			}

			if contrib > remainingToTarget {
				contrib = remainingToTarget
			}

			allocated[i] = contrib
			totalAllocated += contrib

			log.Printf("[DISTRIBUTION] Goal %d (%s, priority %d): calculated %d₽, remaining_to_target %d₽, allocated %d₽",
				goals[i].ID, goals[i].GoalName, goals[i].Priority, (availableForSavings*priorityWeight)/summaryFactorial, remainingToTarget, contrib)
		}
	}

	remainingToDistribute := availableForSavings - totalAllocated
	log.Printf("[DISTRIBUTION] Phase 1 total allocated: %d₽, remaining to distribute: %d₽", totalAllocated, remainingToDistribute)

	if remainingToDistribute > 0 {
		eligibleGoals := []int{}
		for i, goal := range goals {
			if goal.Status == "active" {
				remainingToTarget := goal.TargetAmount - goal.CurrentAmount
				if remainingToTarget > allocated[i] { // Есть место для дополнительных средств
					eligibleGoals = append(eligibleGoals, i)
				}
			}
		}

		if len(eligibleGoals) > 0 {
			baseAmount := remainingToDistribute / int64(len(eligibleGoals))
			remainder := remainingToDistribute % int64(len(eligibleGoals))

			for j, goalIndex := range eligibleGoals {
				extraAmount := baseAmount
				if j < int(remainder) {
					extraAmount++
				}

				remainingToTarget := goals[goalIndex].TargetAmount - goals[goalIndex].CurrentAmount - allocated[goalIndex]
				if extraAmount > remainingToTarget {
					extraAmount = remainingToTarget
				}

				allocated[goalIndex] += extraAmount
				totalAllocated += extraAmount

				log.Printf("[DISTRIBUTION] Goal %d (%s): +%d₽ extra (remaining limit %d₽), total now %d₽",
					goals[goalIndex].ID, goals[goalIndex].GoalName, extraAmount, remainingToTarget, allocated[goalIndex])
			}
		}
	}

	log.Printf("[DISTRIBUTION] Final allocation total: %d₽ (available: %d₽)", totalAllocated, availableForSavings)

	for i := range goals {
		if goals[i].Status == "active" {
			contrib := allocated[i]
			if contrib == 0 && availableForSavings > 0 {
				contrib = 1 // Минимум для активной цели
			}

			goals[i].MonthlyContrib = contrib
			goals[i].MonthlyBudgetLimit = contrib

			currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)

			if !goals[i].MonthStarted.Valid || goals[i].MonthStarted.Time != currentMonth {
				goals[i].MonthStarted.Valid = true
				goals[i].MonthStarted.Time = currentMonth
				goals[i].MonthlyAccumulated = 0
			}

			remaining := goals[i].TargetAmount - goals[i].CurrentAmount
			if remaining > 0 {
				monthsNeeded := (remaining + goals[i].MonthlyContrib - 1) / goals[i].MonthlyContrib
				if monthsNeeded == 0 {
					monthsNeeded = 1
				}
				goals[i].TargetDate = time.Now().AddDate(0, int(monthsNeeded), 0)
			}

			err = s.goalRepo.UpdateGoal(ctx, &goals[i])
			if err != nil {
				log.Printf("Failed to update goal %d: %v", goals[i].ID, err)
			}
		}
	}

	return goals, nil
}

func (s *FinanceService) CreateUser(ctx context.Context, telegramID int64, username string, authToken string) (*models.User, error) {
	user := &models.User{
		TelegramID: telegramID,
		Username:   username,
		AuthToken:  authToken,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	user, err := s.userRepo.CreateUser(ctx, user)
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

func (s *FinanceService) CreateIncomeWithFrequency(ctx context.Context, telegramID int64, name string, amount int64, frequency string, recurringDay int, nextPayDate time.Time) (*models.Income, error) {
	return s.CreateIncomeWithFrequencyAndHour(ctx, telegramID, name, amount, frequency, recurringDay, 18, nextPayDate)
}

func (s *FinanceService) CreateIncomeWithFrequencyAndHour(ctx context.Context, telegramID int64, name string, amount int64, frequency string, recurringDay int, notificationHour int, nextPayDate time.Time) (*models.Income, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	income, err := s.incomeRepo.CreateIncomeWithFrequency(ctx, user.ID, name, amount, frequency, recurringDay, notificationHour, nextPayDate)
	if err != nil {
		log.Printf("Failed to create income: %v", err)
		return nil, err
	}

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	if err != nil {
		log.Printf("Failed to distribute funds after creating income: %v", err)
	}

	log.Printf("Income created: %s, Amount=%d, Frequency=%s, RecurringDay=%d, NotificationHour=%d", name, amount, frequency, recurringDay, notificationHour)
	return income, nil
}

func (s *FinanceService) GetUserIncomes(ctx context.Context, telegramID int64) ([]models.Income, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.incomeRepo.GetUserIncomes(ctx, user.ID)
}

func (s *FinanceService) GetIncomesByPayDate(ctx context.Context, payDate time.Time) ([]models.Income, error) {
	return s.incomeRepo.GetIncomesByPayDate(ctx, payDate)
}

func (s *FinanceService) GetIncomesByPayDateAndHour(ctx context.Context, payDate time.Time, hour int) ([]models.Income, error) {
	return s.incomeRepo.GetIncomesByPayDateAndHour(ctx, payDate, hour)
}

func (s *FinanceService) UpdateIncomeNextPayDate(ctx context.Context, incomeID int64, nextPayDate time.Time) error {
	return s.incomeRepo.UpdateIncomeNextPayDate(ctx, incomeID, nextPayDate)
}

func (s *FinanceService) GetUserIncomeByID(ctx context.Context, telegramID int64, incomeID int64) (*models.Income, error) {
	log.Printf("GetUserIncomeByID: telegramID=%d, incomeID=%d", telegramID, incomeID)

	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		log.Printf("GetUserIncomeByID: user not found for telegramID %d: %v", telegramID, err)
		return nil, fmt.Errorf("user not found: %w", err)
	}

	log.Printf("GetUserIncomeByID: found user ID %d", user.ID)

	income, err := s.incomeRepo.GetIncomeByID(ctx, incomeID)
	if err != nil {
		log.Printf("GetUserIncomeByID: income not found for incomeID %d: %v", incomeID, err)
		return nil, fmt.Errorf("income not found: %w", err)
	}

	log.Printf("GetUserIncomeByID: found income ID %d, belongs to user %d", income.ID, income.UserID)

	if income.UserID != user.ID {
		log.Printf("GetUserIncomeByID: income %d belongs to user %d, but expected user %d", incomeID, income.UserID, user.ID)
		return nil, fmt.Errorf("income does not belong to user")
	}

	log.Printf("GetUserIncomeByID: verification passed, returning income")
	return income, nil
}

func (s *FinanceService) DeleteIncome(ctx context.Context, telegramID int64, incomeID int64) error {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	income, err := s.incomeRepo.GetIncomeByID(ctx, incomeID)
	if err != nil {
		return fmt.Errorf("income not found: %w", err)
	}

	if income.UserID != user.ID {
		return fmt.Errorf("income does not belong to user")
	}

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	if err != nil {
		log.Printf("Failed to distribute funds after deleting income: %v", err)
	}

	return s.incomeRepo.DeleteIncome(ctx, incomeID)
}

func (s *FinanceService) CreateExpense(ctx context.Context, telegramID int64, name string, amount int64) (*models.Expense, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	expense, err := s.expenseRepo.CreateExpense(ctx, user.ID, name, amount)
	if err != nil {
		log.Printf("Failed to create expense: %v", err)
		return nil, err
	}

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	if err != nil {
		log.Printf("Failed to distribute funds after creating expense: %v", err)
	}

	log.Printf("Expense created: %s, Amount=%d", name, amount)
	return expense, nil
}

func (s *FinanceService) GetUserExpenses(ctx context.Context, telegramID int64) ([]models.Expense, error) {
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

func (s *FinanceService) CreateGoal(ctx context.Context, telegramID int64, goalName string, targetAmount int64, priority int) (*models.SavingsGoal, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	targetDate := time.Now().AddDate(0, 1, 0)

	goal, err := s.goalRepo.CreateGoal(
		ctx,
		user.ID,
		goalName,
		targetAmount,
		0,
		targetDate,
		priority,
	)

	if err != nil {
		log.Printf("Failed to create goal: %v", err)
		return nil, err
	}

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	if err != nil {
		log.Printf("Failed to distribute funds after creating goal: %v", err)
	}

	goal, err = s.goalRepo.GetGoalByID(ctx, goal.ID)
	if err != nil {
		log.Printf("Failed to refresh goal: %v", err)
		return goal, nil
	}

	log.Printf("Goal created: %s, Target=%d, MonthlyContrib=%d, Priority=%d, TargetDate=%s",
		goalName, targetAmount, goal.MonthlyContrib, priority, goal.TargetDate.Format("02.01.2006"))
	return goal, nil
}

func (s *FinanceService) reindexGoalPriorities(ctx context.Context, userID int64, newPriority int) error {
	goals, err := s.goalRepo.GetUserGoals(ctx, userID)
	if err != nil {
		return err
	}

	for i := range goals {
		if goals[i].Priority >= newPriority && goals[i].Status == "active" {
			goals[i].Priority++
			err = s.goalRepo.UpdateGoal(ctx, &goals[i])
			if err != nil {
				log.Printf("Failed to update goal priority: %v", err)
			}
		}
	}

	return nil
}

func (s *FinanceService) GetUserGoals(ctx context.Context, telegramID int64) ([]models.SavingsGoal, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.goalRepo.GetUserGoals(ctx, user.ID)
}

func (s *FinanceService) GetUserActiveGoalsByTelegramID(ctx context.Context, telegramID int64) ([]models.SavingsGoal, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return s.goalRepo.GetUserActiveGoals(ctx, user.ID)
}

func (s *FinanceService) GetUserGoalByID(ctx context.Context, telegramID int64, goalID int64) (*models.SavingsGoal, error) {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	if goal.UserID != user.ID {
		return nil, fmt.Errorf("goal does not belong to user")
	}

	return goal, nil
}

func (s *FinanceService) ContributeToGoal(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	return s.ContributeToGoalWithMonthlyTracking(ctx, goalID, amount)
}

func (s *FinanceService) WithdrawFromGoal(ctx context.Context, goalID int64, amount int64) (*models.SavingsGoal, error) {
	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	todayDate := time.Now()
	monthStartDate := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	var goalMonthStart time.Time
	if goal.MonthStarted.Valid {
		goalMonthStart = time.Date(goal.MonthStarted.Time.Year(), goal.MonthStarted.Time.Month(), goal.MonthStarted.Time.Day(), 0, 0, 0, 0, time.UTC)
	}

	if !goal.MonthStarted.Valid || goalMonthStart != monthStartDate {
		goal.MonthlyAccumulated = 0
		goal.MonthStarted.Valid = true
		goal.MonthStarted.Time = monthStartDate
	}

	if goal.CurrentAmount < amount {
		goal.CurrentAmount = 0
	} else {
		goal.CurrentAmount -= amount
	}

	if goal.MonthlyAccumulated >= amount {
		goal.MonthlyAccumulated -= amount
	} else {
		goal.MonthlyAccumulated = 0
	}

	currentMonth := time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthlyContribRecord, err := s.monthlyContribRepo.GetContributionByUserGoalMonth(ctx, goal.UserID, goalID, currentMonth)

	if err != nil || monthlyContribRecord == nil {
		_, createErr := s.monthlyContribRepo.CreateContribution(ctx, goal.UserID, goalID, currentMonth, goal.MonthlyAccumulated)
		if createErr != nil {
			log.Printf("[WITHDRAW] Failed to create monthly contribution after withdrawal: %v", createErr)
		}
	} else {
		monthlyContribRecord.AmountContributed = goal.MonthlyAccumulated
		if updateErr := s.monthlyContribRepo.UpdateContribution(ctx, monthlyContribRecord); updateErr != nil {
			log.Printf("[WITHDRAW] Failed to update monthly contribution after withdrawal: %v", updateErr)
		}
	}

	if goal.Status == "completed" && goal.CurrentAmount < goal.TargetAmount {
		goal.Status = "active"
	}

	err = s.goalRepo.UpdateGoal(ctx, goal)
	if err != nil {
		return nil, err
	}

	log.Printf("Withdrew %d from goal %d, new amount: %d, monthly: %d", amount, goalID, goal.CurrentAmount, goal.MonthlyAccumulated)
	return goal, nil
}

func (s *FinanceService) DeleteExpense(ctx context.Context, telegramID int64, expenseID int64) error {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	expense, err := s.expenseRepo.GetExpenseByID(ctx, expenseID)
	if err != nil {
		return fmt.Errorf("expense not found: %w", err)
	}

	if expense.UserID != user.ID {
		return fmt.Errorf("expense does not belong to user")
	}

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	if err != nil {
		log.Printf("Failed to distribute funds after deleting expense: %v", err)
	}

	return s.expenseRepo.DeleteExpense(ctx, expenseID)
}

func (s *FinanceService) DeleteGoal(ctx context.Context, telegramID int64, goalID int64) error {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	goal, err := s.goalRepo.GetGoalByID(ctx, goalID)
	if err != nil {
		return fmt.Errorf("goal not found: %w", err)
	}

	if goal.UserID != user.ID {
		return fmt.Errorf("goal does not belong to user")
	}

	s.reindexDeletedGoalPriorities(ctx, user.ID, goal.Priority)

	err = s.goalRepo.DeleteGoal(ctx, goalID)
	if err != nil {
		return err
	}

	_, err = s.DistributeFundsToGoals(ctx, telegramID)
	return err
}

// переиндексирует приоритеты после удаления цели
func (s *FinanceService) reindexDeletedGoalPriorities(ctx context.Context, userID int64, deletedPriority int) error {
	goals, err := s.goalRepo.GetUserGoals(ctx, userID)
	if err != nil {
		return err
	}

	for i := range goals {
		if goals[i].Priority > deletedPriority && goals[i].Status == "active" {
			goals[i].Priority--
			err = s.goalRepo.UpdateGoal(ctx, &goals[i])
			if err != nil {
				log.Printf("Failed to update goal priority after deletion: %v", err)
			}
		}
	}

	return nil
}

func (s *FinanceService) CreateMonthlyContribution(ctx context.Context, userID, goalID int64, month time.Time, amount int64) (*models.MonthlyContribution, error) {
	return s.monthlyContribRepo.CreateContribution(ctx, userID, goalID, month, amount)
}

func (s *FinanceService) GetMonthlyContribution(ctx context.Context, userID, goalID int64, month time.Time) (*models.MonthlyContribution, error) {
	return s.monthlyContribRepo.GetContributionByUserGoalMonth(ctx, userID, goalID, month)
}

func (s *FinanceService) UpdateMonthlyContribution(ctx context.Context, contribution *models.MonthlyContribution) error {
	return s.monthlyContribRepo.UpdateContribution(ctx, contribution)
}

func (s *FinanceService) GetMonthlyContributions(ctx context.Context, userID int64, month time.Time) ([]models.MonthlyContribution, error) {
	return s.monthlyContribRepo.GetUserContributionsByMonth(ctx, userID, month)
}

func (s *FinanceService) GetUserByID(ctx context.Context, userID int64) (*models.User, error) {
	return s.userRepo.GetUserByID(ctx, userID)
}

type BotAPI interface {
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
}

func (s *FinanceService) TestPaydayNotification(bot BotAPI, ctx context.Context, telegramID int64, incomeID int64) error {
	user, err := s.userRepo.GetUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	income, err := s.incomeRepo.GetIncomeByID(ctx, incomeID)
	if err != nil {
		return fmt.Errorf("income not found: %w", err)
	}

	if income.UserID != user.ID {
		return fmt.Errorf("income does not belong to user")
	}

	// Отправляем тестовое уведомление - справку о тесте
	now := time.Now()
	dateStr := now.Format("02.01.2006")

	testMsg := tgbotapi.NewMessage(telegramID, fmt.Sprintf(
		"🧪 ТЕСТОВОЕ УВЕДОМЛЕНИЕ ЗАПУЩЕНО!\n\n"+
			"💰 Сегодня: %s\n"+
			"🎯 День получения: %s\n"+
			"Сумма: %d₽\n\n"+
			"⚠️ Ожидайте интерактивное меню ниже...",
		dateStr, income.Name, income.Amount,
	))

	_, err = bot.Send(testMsg)
	if err != nil {
		return fmt.Errorf("failed to send test start message: %w", err)
	}

	// меню точно как настоящее уведомление
	s.showTestPaydayMenu(bot, ctx, telegramID, incomeID, income.Name, income.Amount)

	return nil
}

func (s *FinanceService) showTestPaydayMenu(bot BotAPI, ctx context.Context, telegramID int64, incomeID int64, incomeName string, incomeAmount int64) {
	goals, err := s.GetUserActiveGoalsByTelegramID(ctx, telegramID)
	if err != nil {
		log.Printf("❌ Failed to get goals for test payday: %v", err)
		return
	}

	now := time.Now()
	dateStr := now.Format("02.01.2006")

	if len(goals) == 0 {
		msg := fmt.Sprintf(
			"💰 ТЕСТОВОЕ УВЕДОМЛЕНИЕ\n(не обновляет расписание)\n\n"+
				"💰 Сегодня: %s\n"+
				"🎯 День дохода: %s\n"+
				"Сумма: %d₽\n\n"+
				"ℹ️ У вас нет активных целей для накопления",
			dateStr, incomeName, incomeAmount,
		)

		testMsg := tgbotapi.NewMessage(telegramID, msg)
		bot.Send(testMsg)
		return
	}

	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthlyContributions, err := s.GetMonthlyContributions(ctx, telegramID, currentMonth)
	if err != nil {
		log.Printf("Failed to get monthly contributions for test: %v", err)
		monthlyContributions = make([]models.MonthlyContribution, 0)
	}

	contributedMap := make(map[int64]int64)
	for _, contrib := range monthlyContributions {
		contributedMap[contrib.GoalID] += contrib.AmountContributed
	}

	recommendedMap := s.calculateTestSmartPaydayRecommendations(incomeAmount, goals, contributedMap)

	totalRecommended := int64(0)
	for _, amount := range recommendedMap {
		totalRecommended += amount
	}

	text := fmt.Sprintf(
		"🧪 ТЕСТОВОЕ УВЕДОМЛЕНИЕ\n💰 Сегодня: %s\n\n"+
			"🎯 День поступления: %s\n"+
			"Сумма: %d₽\n\n"+
			"📈 Рекомендуем отложить: %d₽\n"+
			"(из этого поступления)\n\n"+
			"ИНТЕРАКТИВНОЕ ТЕСТИРОВАНИЕ:\n"+
			"• Нажмите на цель ниже\n"+
			"• Введите сумму\n"+
			"• Возвратитесь к этому меню\n"+
			"• Повторите до завершения\n\n",
		dateStr, incomeName, incomeAmount, totalRecommended,
	)

	for i, goal := range goals {
		remaining := goal.TargetAmount - goal.CurrentAmount
		if remaining < 0 {
			remaining = 0
		}

		progress := int64(0)
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount * 100) / goal.TargetAmount
		}

		recommended := recommendedMap[goal.ID]

		text += fmt.Sprintf(
			"%d. %s (%d)\n"+
				" Накоплено: %d/%d₽ (%d%%)\n"+
				" Отложить 💰: %d₽ (рекомендация)\n"+
				" До цели: %d₽\n\n",
			i+1, goal.GoalName, goal.Priority,
			goal.CurrentAmount, goal.TargetAmount, progress,
			recommended, remaining,
		)
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, goal := range goals {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("💰 %s (%d)", goal.GoalName, goal.Priority),
			fmt.Sprintf("test_payday_goal_%d_%d", incomeID, goal.ID),
		)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{btn})
	}

	completeBtn := tgbotapi.NewInlineKeyboardButtonData(
		"✅ Завершить тест",
		fmt.Sprintf("test_payday_complete_%d", incomeID),
	)
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{completeBtn})

	msg := tgbotapi.NewMessage(telegramID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("❌ Failed to send test payday notification: %v", err)
	}
}

func (s *FinanceService) calculateTestSmartPaydayRecommendations(incomeAmount int64, goals []models.SavingsGoal, contributedMap map[int64]int64) map[int64]int64 {
	recommendations := make(map[int64]int64)
	remainingIncome := incomeAmount

	sortedGoals := make([]models.SavingsGoal, len(goals))
	copy(sortedGoals, goals)

	for i := 0; i < len(sortedGoals)-1; i++ {
		for j := i + 1; j < len(sortedGoals); j++ {
			if sortedGoals[i].Priority > sortedGoals[j].Priority {
				sortedGoals[i], sortedGoals[j] = sortedGoals[j], sortedGoals[i]
			}
		}
	}

	for _, goal := range sortedGoals {
		if remainingIncome <= 0 {
			break
		}

		alreadyContributed := contributedMap[goal.ID]
		remainingForGoal := goal.MonthlyBudgetLimit - alreadyContributed
		if remainingForGoal < 0 {
			remainingForGoal = 0
		}

		remainingToTarget := goal.TargetAmount - goal.CurrentAmount
		if remainingToTarget < 0 {
			remainingToTarget = 0
		}

		maxRecommend := remainingToTarget
		if maxRecommend > remainingForGoal {
			maxRecommend = remainingForGoal
		}
		if maxRecommend > remainingIncome {
			maxRecommend = remainingIncome
		}

		if maxRecommend > incomeAmount/2 {
			maxRecommend = incomeAmount / 2
		}

		if remainingToTarget <= 5000 {
			maxRecommend = remainingToTarget
		}

		recommendations[goal.ID] = maxRecommend
		remainingIncome -= maxRecommend
	}

	return recommendations
}

func (s *FinanceService) LogIncomeProcessing(ctx context.Context, incomeID, userID int64, processedDate time.Time, incomeAmount int64) (*models.IncomeProcessingLog, error) {
	return s.processingLogRepo.CreateProcessingLog(ctx, incomeID, userID, processedDate, incomeAmount)
}

func (s *FinanceService) IsIncomeProcessedOnDate(ctx context.Context, incomeID int64, processedDate time.Time) (bool, error) {
	return s.processingLogRepo.IsIncomeProcessedOnDate(ctx, incomeID, processedDate)
}

func (s *FinanceService) CompletePaydaySession(ctx context.Context, incomeID int64) error {
	income, err := s.incomeRepo.GetIncomeByID(ctx, incomeID)
	if err != nil {
		return fmt.Errorf("failed to get income: %w", err)
	}

	nextPayDate := s.calculateNextPayDateForIncome(income)
	return s.incomeRepo.UpdateIncomeNextPayDate(ctx, incomeID, nextPayDate)
}

func (s *FinanceService) calculateNextPayDateForIncome(income *models.Income) time.Time {
	now := time.Now()

	switch income.Frequency {
	case "monthly":
		next := time.Date(now.Year(), now.Month()+1, income.RecurringDay, 9, 0, 0, 0, now.Location())
		maxDay := daysInMonth(next.Year(), next.Month())
		if income.RecurringDay > maxDay {
			next = time.Date(next.Year(), next.Month(), maxDay, 9, 0, 0, 0, now.Location())
		}
		return next

	case "weekly":
		daysUntil := (income.RecurringDay - int(now.Weekday()) + 7) % 7
		if daysUntil <= 0 {
			daysUntil = 7 // Если сегодня этот день, то на следующей неделе
		}

		next := now.AddDate(0, 0, daysUntil)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())

	case "biweekly":
		daysUntil := (income.RecurringDay - int(now.Weekday()) + 7) % 7

		if daysUntil == 0 {
			// Сегодня этот день - через 14 дней
			daysUntil = 14
		} else if daysUntil < 0 {
			// Прошли уже этот день на неделе - + полная неделя и дни до дня
			daysUntil = (7 - int(now.Weekday()) + income.RecurringDay + 7)
		} else {
			// Будет на этой неделе - через неделю
			daysUntil = daysUntil + 7
		}

		next := now.AddDate(0, 0, daysUntil)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())

	default:
		return now.AddDate(0, 0, 1)
	}
}

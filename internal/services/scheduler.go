package services

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/repository"
	"log"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Scheduler struct {
	bot              *tgbotapi.BotAPI
	financeService   *FinanceService
	userRepo         *repository.UserRepository
	contributionRepo *repository.MonthlyContributionsRepository
}

func NewScheduler(bot *tgbotapi.BotAPI, financeService *FinanceService, userRepo *repository.UserRepository, contributionRepo *repository.MonthlyContributionsRepository) *Scheduler {
	return &Scheduler{
		bot:              bot,
		financeService:   financeService,
		userRepo:         userRepo,
		contributionRepo: contributionRepo,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	log.Println("Scheduler started, checking every hour...")

	s.checkPayDates(ctx)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return nil

		case <-ticker.C:
			now := time.Now()
			if now.Minute() < 5 {
				currentHour := now.Hour()
				log.Printf("⏰ %02d:00 - Checking for payday notifications...", currentHour)
				s.checkPayDatesForHour(ctx, currentHour)
			}
		}
	}
}

func (s *Scheduler) checkPayDatesForHour(ctx context.Context, hour int) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	log.Printf("Checking pay dates for today at hour %d: %s", hour, today.Format("02.01.2006"))

	incomes, err := s.financeService.GetIncomesByPayDateAndHour(ctx, today, hour)
	if err != nil {
		log.Printf("Failed to get incomes by pay date and hour: %v", err)
		return
	}

	if len(incomes) == 0 {
		log.Printf("No pay dates today at hour %d", hour)
		return
	}

	log.Printf("Found %d income(s) for today at hour %d", len(incomes), hour)

	for _, income := range incomes {
		processed, err := s.financeService.IsIncomeProcessedOnDate(ctx, income.ID, today)
		if err != nil {
			log.Printf("Failed to check if income %d was processed: %v", income.ID, err)
		} else if processed {
			log.Printf("Income %d already processed for %s, skipping", income.ID, today.Format("02.01.2006"))
			continue
		}

		user, err := s.userRepo.GetUserByID(ctx, income.UserID)
		if err != nil {
			log.Printf("Failed to get user %d: %v", income.UserID, err)
			continue
		}

		s.sendPaydayNotification(ctx, income, user.TelegramID)

		_, err = s.financeService.LogIncomeProcessing(ctx, income.ID, income.UserID, today, income.Amount)
		if err != nil {
			log.Printf("Failed to log income processing for income %d: %v", income.ID, err)
		}
	}
}

func (s *Scheduler) checkPayDates(ctx context.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	log.Printf("Checking pay dates for today: %s", today.Format("02.01.2006"))

	incomes, err := s.financeService.GetIncomesByPayDate(ctx, today)
	if err != nil {
		log.Printf("Failed to get incomes by pay date: %v", err)
		return
	}

	if len(incomes) == 0 {
		log.Println("No pay dates today")
		return
	}

	log.Printf("Found %d income(s) for today", len(incomes))

	for _, income := range incomes {
		user, err := s.userRepo.GetUserByID(ctx, income.UserID)
		if err != nil {
			log.Printf("Failed to get user %d: %v", income.UserID, err)
			continue
		}

		s.sendPaydayNotification(ctx, income, user.TelegramID)

		nextPayDate := s.calculateNextPayDate(income.Frequency, income.RecurringDay)
		err = s.financeService.UpdateIncomeNextPayDate(ctx, income.ID, nextPayDate)
		if err != nil {
			log.Printf("Failed to update next pay date for income %d: %v", income.ID, err)
		}
	}
}

func (s *Scheduler) sendPaydayNotification(ctx context.Context, income models.Income, telegramID int64) {
	goals, err := s.financeService.GetUserActiveGoalsByTelegramID(ctx, telegramID)
	if err != nil {
		log.Printf("Failed to get goals for user %d: %v", telegramID, err)
		return
	}

	now := time.Now()
	dateStr := now.Format("02.01.2006")

	if len(goals) == 0 {
		msg := fmt.Sprintf(
			"💰 Сегодня: %s\n\n"+
				"🎯 День получки: %s\n"+
				"Сумма: %d₽\n\n"+
				"ℹ️ У вас нет активных целей для накопления",
			dateStr, income.Name, income.Amount,
		)

		s.sendNotification(telegramID, msg, nil)
		return
	}

	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthlyContributions, err := s.financeService.GetMonthlyContributions(ctx, income.UserID, currentMonth)
	if err != nil {
		log.Printf("Failed to get monthly contributions: %v", err)
		monthlyContributions = make([]models.MonthlyContribution, 0)
	}

	contributedMap := make(map[int64]int64)
	for _, contrib := range monthlyContributions {
		contributedMap[contrib.GoalID] += contrib.AmountContributed
	}

	recommendedMap := s.calculateSmartPaydayRecommendations(income.Amount, goals, contributedMap)

	totalRecommended := int64(0)
	for _, amount := range recommendedMap {
		totalRecommended += amount
	}

	totalMonthlyPlan := int64(0)
	totalAlreadyContributed := int64(0)
	for _, goal := range goals {
		totalMonthlyPlan += goal.MonthlyContrib
		totalAlreadyContributed += contributedMap[goal.ID]
	}

	text := fmt.Sprintf(
		"💰 Сегодня: %s\n\n"+
			"🎯 День поступления: %s\n"+
			"Сумма: %d₽\n\n"+
			"📈 Рекомендуем отложить: %d₽\n"+
			"(из этого поступления)\n\n"+
			"📊 Месячный план: %d/%d₽\n"+
			"(уже отложено / нужно)\n\n",
		dateStr, income.Name, income.Amount,
		totalRecommended,
		totalAlreadyContributed, totalMonthlyPlan,
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
			fmt.Sprintf("payday_goal_%d_%d", income.ID, goal.ID),
		)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{btn})
	}

	completeBtn := tgbotapi.NewInlineKeyboardButtonData(
		"✅ Завершить",
		fmt.Sprintf("payday_complete_%d", income.ID),
	)
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{completeBtn})

	msg := tgbotapi.NewMessage(telegramID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	_, err = s.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send payday notification to %d: %v", telegramID, err)
	} else {
		log.Printf("Sent payday notification to user %d for income: %s (%d₽)", telegramID, income.Name, income.Amount)
	}
}

func (s *Scheduler) calculateSmartPaydayRecommendations(incomeAmount int64, goals []models.SavingsGoal, contributedMap map[int64]int64) map[int64]int64 {
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

func (s *Scheduler) sendNotification(chatID int64, text string, buttons *tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	if buttons != nil {
		msg.ReplyMarkup = buttons
	}

	_, err := s.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send notification to %d: %v", chatID, err)
	}
}

func (s *Scheduler) calculateNextPayDate(frequency string, recurringDay int) time.Time {
	now := time.Now()

	switch frequency {
	case "monthly":
		next := time.Date(now.Year(), now.Month()+1, recurringDay, 9, 0, 0, 0, now.Location())
		maxDay := daysInMonth(next.Year(), next.Month())
		if recurringDay > maxDay {
			next = time.Date(next.Year(), next.Month(), maxDay, 9, 0, 0, 0, now.Location())
		}
		return next

	case "weekly":
		daysUntil := (recurringDay - int(now.Weekday()) + 7) % 7
		if daysUntil <= 0 {
			daysUntil = 7
		}
		next := now.AddDate(0, 0, daysUntil)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())

	case "biweekly":
		daysUntil := (recurringDay - int(now.Weekday()) + 7) % 7

		if daysUntil == 0 {
			daysUntil = 14
		} else if daysUntil < 0 {
			daysUntil = (7 - int(now.Weekday()) + recurringDay + 7)
		} else {
			daysUntil = daysUntil + 7
		}

		next := now.AddDate(0, 0, daysUntil)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())

	default:
		return now.AddDate(0, 0, 1)
	}
}

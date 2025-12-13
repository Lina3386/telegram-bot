package bot_handler

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *BotHandler) calculateNextPayDate(frequency string, recurringDay int) time.Time {
	now := time.Now()
	switch frequency {
	case "monthly":
		next := time.Date(now.Year(), now.Month(), recurringDay, 9, 0, 0, 0, now.Location())
		if next.After(now) {
			return next
		}
		return time.Date(now.Year(), now.Month()+1, recurringDay, 9, 0, 0, 0, now.Location())
	case "weekly":
		daysUntil := (recurringDay - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 && now.Hour() >= 9 {
			daysUntil = 7
		}
		next := now.AddDate(0, 0, daysUntil).Truncate(24 * time.Hour)
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())
	case "biweekly":
		daysUntil := (recurringDay - int(now.Weekday()) + 7) % 7
		next := now.AddDate(0, 0, daysUntil).Truncate(24 * time.Hour)
		if next.Before(now) || (next.Equal(now.Truncate(24*time.Hour)) && now.Hour() >= 9) {
			next = next.AddDate(0, 0, 7)
		}
		return time.Date(next.Year(), next.Month(), next.Day(), 9, 0, 0, 0, next.Location())
	default:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location())
	}
}

func (h *BotHandler) handlePaydayAmountInput(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	text := message.Text
	ctx := context.Background()

	amount, err := strconv.ParseInt(text, 10, 64)
	if err != nil || amount <= 0 {
		h.sendMessage(chatID, "❌ Введите корректное число")
		return
	}

	goalIDStr := h.stateManager.GetTempData(userID, "payday_contributing_goal_id")
	incomeIDStr := h.stateManager.GetTempData(userID, "payday_contributing_income_id")

	if goalIDStr == "" || incomeIDStr == "" {
		h.sendMessage(chatID, "❌ Ошибка: данные о цели не найдены")
		return
	}

	goalID, _ := strconv.ParseInt(goalIDStr, 10, 64)
	incomeID, _ := strconv.ParseInt(incomeIDStr, 10, 64)

	goal, err := h.financeService.ContributeToGoal(ctx, goalID, amount)
	if err != nil {
		log.Printf("Failed to contribute to goal: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при добавлении")
		return
	}

	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	contrib, err := h.financeService.GetMonthlyContribution(ctx, userID, goalID, currentMonth)

	if err != nil || contrib == nil {
		_, err = h.financeService.CreateMonthlyContribution(ctx, userID, goalID, currentMonth, amount)
	} else {
		contrib.AmountContributed += amount
		err = h.financeService.UpdateMonthlyContribution(ctx, contrib)
	}

	if err != nil {
		log.Printf("Failed to save monthly contribution: %v", err)
	}

	fmt.Sprintf("%s: +%d₽ (total: %d₽)", goal.GoalName, amount, goal.CurrentAmount)

	incomes, err := h.financeService.GetUserIncomes(ctx, userID)
	if err != nil {
		log.Printf("Failed to get user incomes: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при возврате к меню дохода")
		return
	}

	incomeName := ""
	incomeAmount := int64(0)
	for _, income := range incomes {
		if income.ID == incomeID {
			incomeName = income.Name
			incomeAmount = income.Amount
			break
		}
	}

	if incomeName == "" {
		h.sendMessage(chatID, "❌ Не найден доход для меню дохода")
		return
	}

	h.stateManager.ClearState(userID)
	h.showPaydayMenu(userID, chatID, incomeID, incomeName, incomeAmount, ctx)
}

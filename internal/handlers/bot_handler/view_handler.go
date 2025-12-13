package bot_handler

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
	"time"
)

func (h *BotHandler) showGoalDetails(userID int64, chatID int64, goalID int64) {
	ctx := context.Background()

	goal, err := h.financeService.GetUserGoalByID(ctx, userID, goalID)
	if err != nil {
		log.Printf("Failed to get goal: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке цели")
		return
	}

	progress := int64(0)
	if goal.TargetAmount > 0 {
		progress = (goal.CurrentAmount * 100) / goal.TargetAmount
	}

	priorityText := []string{"", "🔴 Высший", "🟡 Средний", "🟢 Низкий"}[goal.Priority]
	if priorityText == "" {
		priorityText = "🟡 Средний"
	}

	timeToGoal := h.calculateTimeToGoal(goal.TargetAmount, goal.MonthlyContrib, goal.CurrentAmount)

	statusText := "Активна ✅"
	if goal.Status == "completed" {
		statusText = "Завершена 🎉"
	}

	// Полная информация о цели
	text := fmt.Sprintf(
		"🎯 %s\n\n"+
			"Приоритет: %s (%d)\n"+
			"Статус: %s\n\n"+
			"Целевая сумма: %d₽\n"+
			"Накоплено: %d₽\n"+
			"Прогресс: %d%%\n"+
			"Осталось: %d₽\n\n"+
			"Месячный взнос: %d₽\n"+
			"Время до цели: %s\n"+
			"Дата достижения: %s",
		goal.GoalName,
		priorityText, goal.Priority,
		statusText,
		goal.TargetAmount,
		goal.CurrentAmount,
		progress,
		goal.TargetAmount-goal.CurrentAmount,
		goal.MonthlyContrib,
		timeToGoal,
		goal.TargetDate.Format("02.01.2006"),
	)

	// Кнопки действий
	var buttons [][]tgbotapi.InlineKeyboardButton

	if goal.Status == "active" {
		// Внести, Снять
		contributeBtn := tgbotapi.NewInlineKeyboardButtonData("💰 Внести", fmt.Sprintf("contrib_%d", goal.ID))
		withdrawBtn := tgbotapi.NewInlineKeyboardButtonData("📤 Снять", fmt.Sprintf("withdraw_%d", goal.ID))

		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{contributeBtn, withdrawBtn})
	}

	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить цель", fmt.Sprintf("delete_goal_%d", goal.ID))
	backBtn := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к целям", "back_to_goals")

	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{deleteBtn, backBtn})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

func (h *BotHandler) showPaydayMenu(userID int64, chatID int64, incomeID int64, incomeName string, incomeAmount int64, ctx context.Context) {
	goals, err := h.financeService.GetUserActiveGoalsByTelegramID(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке целей")
		return
	}

	if len(goals) == 0 {
		msg := fmt.Sprintf("💰 Сегодня: %s\n\n%s: %d₽\n\n🎯 У вас нет активных целей для накопления",
			time.Now().Format("02.01.2006"), incomeName, incomeAmount)
		h.sendMessageWithKeyboard(chatID, msg, h.mainMenu())
		return
	}

	contributedMap := make(map[int64]int64)
	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)

	for _, goal := range goals {
		monthlyContribRecord, err := h.financeService.GetMonthlyContribution(ctx, userID, goal.ID, currentMonth)
		if err == nil && monthlyContribRecord != nil {
			contributedMap[goal.ID] = monthlyContribRecord.AmountContributed
			log.Printf("[PAYDAY_MENU] Goal %d (%s): using monthly_contributions %d₽ (goal.MonthlyAccumulated was %d₽)", goal.ID, goal.GoalName, contributedMap[goal.ID], goal.MonthlyAccumulated)
		} else {
			contributedMap[goal.ID] = goal.MonthlyAccumulated
		}
	}

	totalMonthlyPlan := int64(0)
	totalAlreadyContributed := int64(0)
	for _, goal := range goals {
		totalMonthlyPlan += goal.MonthlyContrib
		totalAlreadyContributed += contributedMap[goal.ID]
	}

	text := fmt.Sprintf(
		"💰 Сегодня: %s\n\n"+
			"🎯 День дохода: %s\n"+
			"Сумма: %d₽\n\n"+
			"Нужно отложить в этом месяце:\n"+
			"%d/%d₽\n\n",
		time.Now().Format("02.01.2006"), incomeName, incomeAmount, totalAlreadyContributed, totalMonthlyPlan,
	)

	// Информация по целям
	for i, goal := range goals {
		remaining := goal.TargetAmount - goal.CurrentAmount
		if remaining < 0 {
			remaining = 0
		}

		progress := int64(0)
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount * 100) / goal.TargetAmount
		}

		contributed := contributedMap[goal.ID]

		text += fmt.Sprintf(
			"%d. %s (%d)\n"+
				"   Накоплено: %d/%d₽ (%d%%)\n"+
				"   Отложить: %d/%d₽\n"+
				"   Осталось: %d₽\n\n",
			i+1, goal.GoalName, goal.Priority,
			goal.CurrentAmount, goal.TargetAmount, progress,
			contributed, goal.MonthlyContrib, remaining,
		)
	}

	// Кнопки целей
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, goal := range goals {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("💰 %s (%d)", goal.GoalName, goal.Priority),
			fmt.Sprintf("payday_goal_%d_%d", incomeID, goal.ID),
		)
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{btn})
	}

	completeBtn := tgbotapi.NewInlineKeyboardButtonData(
		"✅ Завершить",
		fmt.Sprintf("payday_complete_%d", incomeID),
	)
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{completeBtn})

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	h.bot.Send(msg)
}

func (h *BotHandler) showPaydayGoalMenu(userID int64, chatID int64, incomeID int64, goalID int64, ctx context.Context) {
	goal, err := h.financeService.GetUserGoalByID(ctx, userID, goalID)
	if err != nil {
		log.Printf("Failed to get goal: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке цели")
		return
	}

	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	contrib, err := h.financeService.GetMonthlyContribution(ctx, userID, goalID, currentMonth)

	contributed := int64(0)
	if err == nil && contrib != nil {
		contributed = contrib.AmountContributed
	}

	remaining := goal.TargetAmount - goal.CurrentAmount
	if remaining < 0 {
		remaining = 0
	}

	text := fmt.Sprintf(
		"🎯 Цель: %s\n"+
			"Приоритет: (%d)\n\n"+
			"Накоплено: %d/%d₽\n"+
			"Осталось накопить: %d₽\n\n"+
			"Можно отложить в этом месяце:\n"+
			"%d/%d₽\n\n"+
			"Всего накоплено: %d₽",
		goal.GoalName, goal.Priority,
		goal.CurrentAmount, goal.TargetAmount,
		remaining,
		contributed, goal.MonthlyContrib,
		goal.CurrentAmount,
	)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", fmt.Sprintf("payday_add_contribution_%d_%d", incomeID, goalID)),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад", fmt.Sprintf("payday_back_%d", incomeID)),
		},
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)

	h.bot.Send(msg)
}

func (h *BotHandler) showGoalDetailsV2(userID int64, chatID int64, goalID int64) {
	ctx := context.Background()

	goal, err := h.financeService.GetUserGoalByID(ctx, userID, goalID)
	if err != nil {
		log.Printf("Failed to get goal: %v", err)
		h.answerCallback("", "❌ Ошибка")
		return
	}

	allGoals, err := h.financeService.GetUserGoals(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
		allGoals = []models.SavingsGoal{}
	}

	monthlyStats, err := h.financeService.GetGoalMonthlyStats(ctx, goalID)
	if err != nil {
		log.Printf("Failed to get monthly stats: %v", err)
		monthlyStats = make(map[string]interface{})
	}

	priorityText := fmt.Sprintf("%d", goal.Priority)
	if goal.Priority == 1 {
		priorityText = "🥇 1 (Наивысший)"
	} else if goal.Priority == 2 {
		priorityText = "🥈 2"
	} else if goal.Priority == 3 {
		priorityText = "🥉 3"
	} else {
		priorityText = fmt.Sprintf("%d", goal.Priority)
	}

	// Статус прогресса
	progress := int64(0)
	if goal.TargetAmount > 0 {
		progress = (goal.CurrentAmount * 100) / goal.TargetAmount
	}

	statusText := "🔄 В процессе"
	if goal.Status == "completed" {
		statusText = "✅ Достигнута"
	} else if goal.Status == "paused" {
		statusText = "⏸️ На паузе"
	}

	// Месячный прогресс
	monthlyAccumulated := int64(0)
	monthlyBudget := int64(0)
	monthlyProgress := int64(0)

	if monthlyStats != nil {
		if val, ok := monthlyStats["monthly_accumulated"].(int64); ok {
			monthlyAccumulated = val
		}
		if val, ok := monthlyStats["monthly_budget_limit"].(int64); ok {
			monthlyBudget = val
		}
		if val, ok := monthlyStats["monthly_progress"].(int64); ok {
			monthlyProgress = val
		}
	}

	// Полная информация о цели
	text := fmt.Sprintf(
		"🎯 <b>%s</b>\n\n"+
			"<b>Приоритет:</b> %s\n"+
			"<b>Статус:</b> %s\n\n"+
			"<b>Целевая сумма:</b> %d₽\n"+
			"<b>Накоплено:</b> %d₽\n"+
			"<b>Прогресс:</b> %d%%\n"+
			"<b>Осталось:</b> %d₽\n\n"+
			"<b>На этот месяц:</b> %d₽ / %d₽ (%d%%)\n"+
			"<b>Дата достижения:</b> %s",
		goal.GoalName,
		priorityText,
		statusText,
		goal.TargetAmount,
		goal.CurrentAmount,
		progress,
		goal.TargetAmount-goal.CurrentAmount,
		monthlyAccumulated, monthlyBudget, monthlyProgress,
		goal.TargetDate.Format("02.01.2006"),
	)

	log.Printf("[GOAL_DETAILS_V2] Goal %d (%s): Target=%d₽, Current=%d₽, Remaining=%d₽, MonthlyAccum=%d₽, MonthlyBudget=%d₽, MonthlyContrib=%d₽",
		goal.ID, goal.GoalName, goal.TargetAmount, goal.CurrentAmount, goal.TargetAmount-goal.CurrentAmount, monthlyAccumulated, monthlyBudget, goal.MonthlyContrib)
	log.Printf("[GOAL_DETAILS_V2] Message text: %s", text)

	// Кнопки действий
	var buttons [][]tgbotapi.InlineKeyboardButton

	if goal.Status == "active" {
		// Внести, Снять
		contributeBtn := tgbotapi.NewInlineKeyboardButtonData("💰 Внести", fmt.Sprintf("contrib_%d", goal.ID))
		withdrawBtn := tgbotapi.NewInlineKeyboardButtonData("📤 Снять", fmt.Sprintf("withdraw_%d", goal.ID))
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{contributeBtn, withdrawBtn})

		// Кнопка изменения приоритета (только если больше одной цели)
		if len(allGoals) > 1 {
			changePriorityBtn := tgbotapi.NewInlineKeyboardButtonData("🔀 Изменить приоритет", fmt.Sprintf("changepriority_%d", goal.ID))
			buttons = append(buttons, []tgbotapi.InlineKeyboardButton{changePriorityBtn})
		}
	}

	// Кнопки удаления и возврата
	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить цель", fmt.Sprintf("delete_goal_%d", goal.ID))
	backBtn := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к целям", "back_to_goals")
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{deleteBtn, backBtn})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	h.bot.Send(msg)
}

func (h *BotHandler) showGoalDetailsV2WithBack(userID int64, chatID int64, goalID int64, backCallback string, backText string) {
	ctx := context.Background()

	goal, err := h.financeService.GetUserGoalByID(ctx, userID, goalID)
	if err != nil {
		log.Printf("Failed to get goal: %v", err)
		h.answerCallback("", "❌ Ошибка")
		return
	}

	allGoals, err := h.financeService.GetUserGoals(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
		allGoals = []models.SavingsGoal{}
	}

	monthlyStats, err := h.financeService.GetGoalMonthlyStats(ctx, goalID)
	if err != nil {
		log.Printf("Failed to get monthly stats: %v", err)
		monthlyStats = make(map[string]interface{})
	}

	priorityText := fmt.Sprintf("%d", goal.Priority)
	if goal.Priority == 1 {
		priorityText = "🥇 1 (Наивысший)"
	} else if goal.Priority == 2 {
		priorityText = "🥈 2"
	} else if goal.Priority == 3 {
		priorityText = "🥉 3"
	} else {
		priorityText = fmt.Sprintf("%d", goal.Priority)
	}

	// Статус прогресса
	progress := int64(0)
	if goal.TargetAmount > 0 {
		progress = (goal.CurrentAmount * 100) / goal.TargetAmount
	}

	statusText := "🔄 В процессе"
	if goal.Status == "completed" {
		statusText = "✅ Достигнута"
	} else if goal.Status == "paused" {
		statusText = "⏸️ На паузе"
	}

	// Месячный прогресс
	monthlyAccumulated := int64(0)
	monthlyBudget := int64(0)
	monthlyProgress := int64(0)

	if monthlyStats != nil {
		if val, ok := monthlyStats["monthly_accumulated"].(int64); ok {
			monthlyAccumulated = val
		}
		if val, ok := monthlyStats["monthly_budget_limit"].(int64); ok {
			monthlyBudget = val
		}
		if val, ok := monthlyStats["monthly_progress"].(int64); ok {
			monthlyProgress = val
		}
	}

	// Полная информация о цели
	text := fmt.Sprintf(
		"🎯 <b>%s</b>\n\n"+
			"<b>Приоритет:</b> %s\n"+
			"<b>Статус:</b> %s\n\n"+
			"<b>Целевая сумма:</b> %d₽\n"+
			"<b>Накоплено:</b> %d₽\n"+
			"<b>Прогресс:</b> %d%%\n"+
			"<b>Осталось:</b> %d₽\n\n"+
			"<b>На этот месяц:</b> %d₽ / %d₽ (%d%%)\n"+
			"<b>Дата достижения:</b> %s",
		goal.GoalName,
		priorityText,
		statusText,
		goal.TargetAmount,
		goal.CurrentAmount,
		progress,
		goal.TargetAmount-goal.CurrentAmount,
		monthlyAccumulated, monthlyBudget, monthlyProgress,
		goal.TargetDate.Format("02.01.2006"),
	)

	incomeIDForPayday := int64(0)
	if strings.HasPrefix(backCallback, "payday_back_") {
		incomeIDStr := strings.TrimPrefix(backCallback, "payday_back_")
		if incomeIDParsed, err := strconv.ParseInt(incomeIDStr, 10, 64); err == nil {
			incomeIDForPayday = incomeIDParsed
		}
	}

	var buttons [][]tgbotapi.InlineKeyboardButton

	if goal.Status == "active" {
		contributeBtn := tgbotapi.NewInlineKeyboardButtonData("💰 Внести", fmt.Sprintf("payday_add_%d_%d", incomeIDForPayday, goalID))
		withdrawBtn := tgbotapi.NewInlineKeyboardButtonData("📤 Снять", fmt.Sprintf("withdraw_%d", goal.ID))
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{contributeBtn, withdrawBtn})

		if len(allGoals) > 1 {
			changePriorityBtn := tgbotapi.NewInlineKeyboardButtonData("🔀 Изменить приоритет", fmt.Sprintf("changepriority_%d", goal.ID))
			buttons = append(buttons, []tgbotapi.InlineKeyboardButton{changePriorityBtn})
		}
	}

	deleteBtn := tgbotapi.NewInlineKeyboardButtonData("🗑️ Удалить цель", fmt.Sprintf("delete_goal_%d", goal.ID))
	backBtn := tgbotapi.NewInlineKeyboardButtonData(backText, backCallback)
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{deleteBtn, backBtn})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	h.bot.Send(msg)
}

func (h *BotHandler) showTestGoalDetailsV2WithBack(userID int64, chatID int64, goalID int64, incomeID int64) {
	ctx := context.Background()

	goal, err := h.financeService.GetUserGoalByID(ctx, userID, goalID)
	if err != nil {
		log.Printf("Failed to get goal: %v", err)
		h.answerCallback("", "❌ Ошибка теста")
		return
	}

	monthlyStats, err := h.financeService.GetGoalMonthlyStats(ctx, goalID)
	if err != nil {
		log.Printf("Failed to get monthly stats: %v", err)
		monthlyStats = make(map[string]interface{})
	}

	priorityText := fmt.Sprintf("%d", goal.Priority)
	if goal.Priority == 1 {
		priorityText = "🥇 1 (Наивысший)"
	} else if goal.Priority == 2 {
		priorityText = "🥈 2"
	} else if goal.Priority == 3 {
		priorityText = "🥉 3"
	} else {
		priorityText = fmt.Sprintf("%d", goal.Priority)
	}

	progress := int64(0)
	if goal.TargetAmount > 0 {
		progress = (goal.CurrentAmount * 100) / goal.TargetAmount
	}

	statusText := "🔄 В процессе"
	if goal.Status == "completed" {
		statusText = "✅ Достигнута"
	} else if goal.Status == "paused" {
		statusText = "⏸️ На паузе"
	}

	monthlyAccumulated := int64(0)
	monthlyBudget := int64(0)
	monthlyProgress := int64(0)

	if monthlyStats != nil {
		if val, ok := monthlyStats["monthly_accumulated"].(int64); ok {
			monthlyAccumulated = val
		}
		if val, ok := monthlyStats["monthly_budget_limit"].(int64); ok {
			monthlyBudget = val
		}
		if val, ok := monthlyStats["monthly_progress"].(int64); ok {
			monthlyProgress = val
		}
	}

	text := fmt.Sprintf(
		"🎯 <b>%s</b> (ТЕСТ)\n\n"+
			"<b>Приоритет:</b> %s\n"+
			"<b>Статус:</b> %s\n\n"+
			"<b>Целевая сумма:</b> %d₽\n"+
			"<b>Накоплено:</b> %d₽\n"+
			"<b>Прогресс:</b> %d%%\n"+
			"<b>Осталось:</b> %d₽\n\n"+
			"<b>На этот месяц:</b> %d₽ / %d₽ (%d%%)\n"+
			"<b>Дата достижения:</b> %s",
		goal.GoalName,
		priorityText,
		statusText,
		goal.TargetAmount,
		goal.CurrentAmount,
		progress,
		goal.TargetAmount-goal.CurrentAmount,
		monthlyAccumulated, monthlyBudget, monthlyProgress,
		goal.TargetDate.Format("02.01.2006"),
	)

	var buttons [][]tgbotapi.InlineKeyboardButton

	if goal.Status == "active" {
		addBtn := tgbotapi.NewInlineKeyboardButtonData("💰 Внести (тест)", fmt.Sprintf("test_payday_add_%d_%d", incomeID, goalID))
		buttons = append(buttons, []tgbotapi.InlineKeyboardButton{addBtn})
	}

	testCompleteBtn := tgbotapi.NewInlineKeyboardButtonData("✅ Завершить тест", fmt.Sprintf("test_payday_complete_%d", incomeID))
	backBtn := tgbotapi.NewInlineKeyboardButtonData("⬅️ Назад к тесту", fmt.Sprintf("test_payday_back_%d", incomeID))
	buttons = append(buttons, []tgbotapi.InlineKeyboardButton{backBtn, testCompleteBtn})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"

	h.bot.Send(msg)
}

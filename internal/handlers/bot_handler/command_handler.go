package bot_handler

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

func (h *BotHandler) handleShowIncomes(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	incomes, err := h.financeService.GetUserIncomes(ctx, userID)
	if err != nil {
		log.Printf("Failed to get incomes: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке доходов")
		return
	}

	text := "💳 Ваши доходы:\n\n"
	var inlineButtons [][]tgbotapi.InlineKeyboardButton

	if len(incomes) > 0 {
		totalIncome, err := h.financeService.CalculateTotalIncome(ctx, userID)
		if err != nil {
			log.Printf("Failed to calculate total income: %v", err)
			totalIncome = 0
		}

		for i, income := range incomes {
			freqText := income.Frequency
			if freqText == "monthly" {
				freqText = "ежемесячно"
			} else if freqText == "weekly" {
				freqText = "еженедельно"
			} else if freqText == "biweekly" {
				freqText = "через неделю"
			}

			dayDesc := fmt.Sprintf("%d число", income.RecurringDay)
			if income.Frequency == "weekly" || income.Frequency == "biweekly" {
				weeks := map[int]string{0: "вс", 1: "пн", 2: "вт", 3: "ср", 4: "чт", 5: "пт", 6: "сб"}
				dayDesc = weeks[income.RecurringDay]
			}

			text += fmt.Sprintf("%d\n💰 %s: %d₽ (%s, %s)\n\n", i+1, income.Name, income.Amount, freqText, dayDesc)

			button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🗑️ Удалить %d", i+1), fmt.Sprintf("delete_income_%d", income.ID))
			inlineButtons = append(inlineButtons, []tgbotapi.InlineKeyboardButton{button})
		}

		text += fmt.Sprintf("\n📈 Общий доход: %d₽\n\n", totalIncome)
	} else {
		text += "У вас нет добавленных доходов\n\n"
	}

	addButton := tgbotapi.NewInlineKeyboardButtonData("➕ Добавить доход", "add_income")
	inlineButtons = append(inlineButtons, []tgbotapi.InlineKeyboardButton{addButton})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineButtons...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	_, err = h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send income list: %v", err)
	}
}

func (h *BotHandler) handleShowExpenses(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	expenses, err := h.financeService.GetUserExpenses(ctx, userID)
	if err != nil {
		log.Printf("Failed to get expenses: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке расходов")
		return
	}

	text := "💰 Ваши расходы:\n\n"
	var inlineButtons [][]tgbotapi.InlineKeyboardButton
	totalExpense := int64(0)

	if len(expenses) > 0 {
		for i, expense := range expenses {
			text += fmt.Sprintf("%d. 📌 %s: %d₽\n", i+1, expense.Name, expense.Amount)
			totalExpense += expense.Amount

			button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🗑️ Удалить #%d", i+1), fmt.Sprintf("delete_expense_%d", expense.ID))
			inlineButtons = append(inlineButtons, []tgbotapi.InlineKeyboardButton{button})
		}

		text += fmt.Sprintf("\n📉 Общие расходы: %d₽\n\n", totalExpense)
	} else {
		text += "У вас нет добавленных расходов\n\n"
	}

	addButton := tgbotapi.NewInlineKeyboardButtonData("➕ Добавить расход", "add_expense")
	inlineButtons = append(inlineButtons, []tgbotapi.InlineKeyboardButton{addButton})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineButtons...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	_, err = h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send expense list: %v", err)
	}
}

func (h *BotHandler) handleShowGoals(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	_, err := h.financeService.DistributeFundsToGoals(ctx, userID)
	if err != nil {
		log.Printf("Failed to redistribute funds on goals view: %v", err)
		// Продолжаем показывать цели даже при ошибке перерасчета
	}

	goals, err := h.financeService.GetUserGoals(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке целей")
		return
	}

	text := "🍀 Ваши цели сбережения:\n\n"

	for i := 0; i < len(goals)-1; i++ {
		for j := i + 1; j < len(goals); j++ {
			if goals[i].Priority > goals[j].Priority {
				goals[i], goals[j] = goals[j], goals[i]
			}
		}
	}

	var inlineButtons [][]tgbotapi.InlineKeyboardButton

	if len(goals) > 0 {
		for _, goal := range goals {
			if goal.Status == "active" {
				progress := int64(0)
				if goal.TargetAmount > 0 {
					progress = (goal.CurrentAmount * 100) / goal.TargetAmount
				}

				priorityStr := ""
				if len(goals) > 1 {
					if goal.Priority == 1 {
						priorityStr = " 🥇"
					} else if goal.Priority == 2 {
						priorityStr = " 🥈"
					} else if goal.Priority == 3 {
						priorityStr = " 🥉"
					} else {
						priorityStr = fmt.Sprintf(" (%d)", goal.Priority)
					}
				}

				monthlyContrib := goal.MonthlyContrib
				if monthlyContrib == 0 {
					monthlyContrib = goal.MonthlyBudgetLimit
				}

				text += fmt.Sprintf(
					"• %s%s: %d₽ / %d₽ (%d%%)\n",
					goal.GoalName, priorityStr, goal.CurrentAmount, goal.TargetAmount, progress,
				)

				btn := tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("ꪜ %s", goal.GoalName),
					fmt.Sprintf("select_goal_%d", goal.ID),
				)
				inlineButtons = append(inlineButtons, []tgbotapi.InlineKeyboardButton{btn})
			}
		}
	}

	if len(goals) == 0 {
		text += "У вас пока нет целей.\n\nНажмите ➕ чтобы создать цель"
	}

	createBtn := tgbotapi.NewInlineKeyboardButtonData("➕ Создать цель", "create_goal")
	inlineButtons = append(inlineButtons, []tgbotapi.InlineKeyboardButton{createBtn})

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineButtons...)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	h.bot.Send(msg)
}

func (h *BotHandler) handleShowStats(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	totalIncome, err := h.financeService.CalculateTotalIncome(ctx, userID)
	if err != nil {
		log.Printf("Failed to calculate total income: %v", err)
	}

	totalExpense, err := h.financeService.CalculateTotalExpense(ctx, userID)
	if err != nil {
		log.Printf("Failed to calculate total expense: %v", err)
	}

	availableForSavings, err := h.financeService.CalculateAvailableForSavings(ctx, userID)
	if err != nil {
		log.Printf("Failed to calculate available for savings: %v", err)
	}

	goals, err := h.financeService.GetUserActiveGoalsByTelegramID(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
	}

	text := fmt.Sprintf(
		"📊 Ваша финансовая статистика:\n\n"+
			"💰 Общий доход: %d₽\n"+
			"💸 Общие расходы: %d₽\n"+
			"🎯 Доступно для сбережений: %d₽\n",
		totalIncome, totalExpense, availableForSavings,
	)

	if len(goals) > 0 {
		text += "\n🎯 Цели накопления:\n\n"
		totalSaved := int64(0)
		totalMonthlyContrib := int64(0)

		for _, goal := range goals {
			progress := int64(0)
			if goal.TargetAmount > 0 {
				progress = (goal.CurrentAmount * 100) / goal.TargetAmount
			}
			remaining := goal.TargetAmount - goal.CurrentAmount
			if remaining < 0 {
				remaining = 0
			}

			text += fmt.Sprintf(
				"🎯 %s\n"+
					"   Накоплено: %d₽ / %d₽ (%d%%)\n"+
					"   Копится в месяц: %d₽\n"+
					"   Осталось: %d₽\n\n",
				goal.GoalName, goal.CurrentAmount, goal.TargetAmount, progress,
				goal.MonthlyContrib, remaining,
			)

			totalSaved += goal.CurrentAmount
			totalMonthlyContrib += goal.MonthlyContrib
		}

		text += fmt.Sprintf(
			"📊 Итого:\n"+
				"   Всего накоплено: %d₽\n"+
				"   Всего копится в месяц: %d₽\n",
			totalSaved, totalMonthlyContrib,
		)
	}

	h.sendMessageWithKeyboard(chatID, text, h.mainMenu())
}

func (h *BotHandler) handleTestPaydayCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	args := strings.Fields(message.Text)
	if len(args) < 2 {
		h.sendMessage(chatID, "❌ Использование: /testpayday [порядковый_номер_дохода]\n\nСначала посмотрите список своих доходов (номер 1,2,3...)")
		return
	}

	incomeIndexStr := args[1]
	incomeIndex, err := strconv.Atoi(incomeIndexStr)
	if err != nil || incomeIndex < 1 {
		h.sendMessage(chatID, "❌ Номер дохода должен быть числом от 1")
		return
	}

	// список доходов пользователя
	incomes, err := h.financeService.GetUserIncomes(ctx, userID)
	if err != nil {
		h.sendMessage(chatID, "❌ Ошибка при загрузке доходов")
		return
	}

	if len(incomes) < incomeIndex {
		h.sendMessage(chatID, fmt.Sprintf("❌ Номер дохода должен быть от 1 до %d", len(incomes)))
		return
	}

	// доход по порядковому номеру
	income := incomes[incomeIndex-1]
	incomeID := income.ID

	err = h.financeService.TestPaydayNotification(h.bot, ctx, userID, incomeID)
	if err != nil {
		h.sendMessage(chatID, fmt.Sprintf("❌ Ошибка: %v", err))
		return
	}

	h.sendMessage(chatID, "✅ Тестовое уведомление отправлено!")
}

package handlers

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
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

	if len(incomes) == 0 {
		h.sendMessageWithKeyboard(chatID, "📊 У вас нет добавленных доходов", h.mainMenu())
		return
	}

	text := "📊 Ваши доходы:\n\n"
	totalIncome := int64(0)
	for _, income := range incomes {
		text += fmt.Sprintf("💰 %s: %d₽ (дата: %d число)\n", income.Name, income.Amount, income.RecurringDay)
		totalIncome += income.Amount
	}
	text += fmt.Sprintf("\n📈 Общий доход: %d₽", totalIncome)

	h.sendMessageWithKeyboard(chatID, text, h.mainMenu())
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

	if len(expenses) == 0 {
		h.sendMessageWithKeyboard(chatID, "💰 У вас нет добавленных расходов", h.mainMenu())
		return
	}

	text := "💰 Ваши расходы:\n\n"
	totalExpense := int64(0)
	for _, expense := range expenses {
		text += fmt.Sprintf("📌 %s: %d₽\n", expense.Name, expense.Amount)
		totalExpense += expense.Amount
	}
	text += fmt.Sprintf("\n📉 Общие расходы: %d₽", totalExpense)

	h.sendMessageWithKeyboard(chatID, text, h.mainMenu())
}

func (h *BotHandler) handleShowGoals(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	goals, err := h.financeService.GetUserGoals(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при загрузке целей")
		return
	}

	if len(goals) == 0 {
		h.stateManager.SetState(userID, state.StateCreatingGoal)
		h.sendMessage(chatID, "🎯 У вас нет целей.\n\nВведите название новой цели:")
		return
	}

	text := "🎯 Ваши цели:\n\n"
	for _, goal := range goals {
		progress := (goal.CurrentAmount * 100) / goal.TargetAmount
		text += fmt.Sprintf(
			"🎯 %s\nЦель: %d₽ | Собрано: %d₽ (%d%%)\nДата: %s | Статус: %s\n\n",
			goal.GoalName, goal.TargetAmount, goal.CurrentAmount, progress,
			goal.TargetDate.Format("02.01.2006"), goal.Status,
		)
	}

	h.sendMessageWithKeyboard(chatID, text, h.mainMenu())
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

	text := fmt.Sprintf(
		"📈 Ваша финансовая статистика:\n\n"+
			"💰 Общий доход: %d₽\n"+
			"💸 Общие расходы: %d₽\n"+
			"🎯 Доступно для сбережений: %d₽\n",
		totalIncome, totalExpense, availableForSavings,
	)

	h.sendMessageWithKeyboard(chatID, text, h.mainMenu())
}

func (h *BotHandler) HandleCallback(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	callbackData := query.Data

	log.Printf("Callback from user %d: %s", userID, callbackData)

	// Разбираем callback данные
	parts := strings.Split(callbackData, "_")
	if len(parts) < 2 {
		h.answerCallback(query.ID, "❌ Неизвестное действие")
		return
	}

	action := parts[0]

	switch action {
	case "add_income":
		h.stateManager.SetState(userID, state.StateAddingIncome)
		h.sendMessage(chatID, "Введите название дохода:")
		h.answerCallback(query.ID, "✅ Введите данные")

	case "add_expense":
		h.stateManager.SetState(userID, state.StateAddingExpense)
		h.sendMessage(chatID, "Введите название расхода:")
		h.answerCallback(query.ID, "✅ Введите данные")

	case "create_goal":
		h.stateManager.SetState(userID, state.StateCreatingGoal)
		h.sendMessage(chatID, "Введите название цели:")
		h.answerCallback(query.ID, "✅ Введите данные")

	default:
		h.answerCallback(query.ID, "❓ Неизвестное действие")
	}
}

func (h *BotHandler) mainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить доход"),
			tgbotapi.NewKeyboardButton("📊 Мои доходы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💰 Мои расходы"),
			tgbotapi.NewKeyboardButton("🎯 Цели"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📈 Статистика"),
		),
	)
}

func (h *BotHandler) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message to %d: %v", chatID, err)
		return err
	}
	return nil
}

func (h *BotHandler) sendMessageWithKeyboard(
	chatID int64,
	text string,
	keyboard tgbotapi.ReplyKeyboardMarkup,
) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message with keyboard to %d: %v", chatID, err)
		return err
	}
	return nil
}

func (h *BotHandler) answerCallback(callbackQueryID, text string) {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	h.bot.Request(callback)
}

package bot_handler

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"strings"
)

func (h *BotHandler) HandleCallback(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	callbackData := query.Data
	ctx := context.Background()

	log.Printf("Callback from user %d: %s", userID, callbackData)

	if strings.HasPrefix(callbackData, "payday_") {
		h.handlePaydayCallbacks(query)
		return
	}
	if strings.HasPrefix(callbackData, "test_payday_") {
		h.handleTestPaydayCallbacks(query)
		return
	}

	shouldDeleteMessage := true
	switch callbackData {
	case "contrib", "contribute", "withdraw", "changepriority":
		// Эти кнопки ведут к вводу текста, не меняют экран - не удаляем
		shouldDeleteMessage = false
	}

	if shouldDeleteMessage {
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		_, err := h.bot.Request(deleteMsg)
		if err != nil {
			log.Printf("Failed to delete original message: %v", err)
		}
	}

	switch callbackData {
	case "add_income":
		h.stateManager.SetState(userID, state.StateAddingIncome)
		h.sendMessage(chatID, "Введите название дохода:")
		h.answerCallback(query.ID, "✅ Введите данные")
		return

	case "add_expense":
		h.stateManager.SetState(userID, state.StateAddingExpense)
		h.sendMessage(chatID, "Введите название расхода:")
		h.answerCallback(query.ID, "✅ Введите данные")
		return

	case "create_goal":
		h.stateManager.SetState(userID, state.StateCreatingGoal)
		h.sendMessage(chatID, "Введите название цели:")
		h.answerCallback(query.ID, "✅ Введите данные")
		return

	case "back_to_goals":
		h.answerCallback(query.ID, "✅")
		h.handleShowGoals(&tgbotapi.Message{
			From: &tgbotapi.User{ID: userID},
			Chat: &tgbotapi.Chat{ID: chatID},
		})
		return
	}

	if strings.HasPrefix(callbackData, "select_goal_") {
		goalIDStr := strings.TrimPrefix(callbackData, "select_goal_")
		goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}

		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		_, err = h.bot.Request(deleteMsg)
		if err != nil {
			log.Printf("Failed to delete original message: %v", err)
		}

		h.showGoalDetailsV2(userID, chatID, goalID)
		return
	}

	parts := strings.SplitN(callbackData, "_", 2)
	if len(parts) < 1 {
		h.answerCallback(query.ID, "❌ Неизвестное действие")
		return
	}

	action := parts[0]
	params := ""
	if len(parts) > 1 {
		params = parts[1]
	}

	switch action {

	case "delete":
		if params == "" {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		resourceParts := strings.SplitN(params, "_", 2)
		if len(resourceParts) < 2 {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		resourceType := resourceParts[0]
		resourceID := resourceParts[1]

		switch resourceType {
		case "income":
			incomeID, err := strconv.ParseInt(resourceID, 10, 64)
			if err != nil {
				h.answerCallback(query.ID, "❌ Ошибка")
				return
			}
			err = h.financeService.DeleteIncome(ctx, userID, incomeID)
			if err != nil {
				log.Printf("Failed to delete income: %v", err)
				h.answerCallback(query.ID, "❌ Ошибка при удалении")
				return
			}
			fmt.Sprintf("income_id: %d", incomeID)
			h.answerCallback(query.ID, "✅ Доход удален")
			h.handleShowIncomes(&tgbotapi.Message{From: &tgbotapi.User{ID: userID}, Chat: &tgbotapi.Chat{ID: chatID}})

		case "expense":
			expenseID, err := strconv.ParseInt(resourceID, 10, 64)
			if err != nil {
				h.answerCallback(query.ID, "❌ Ошибка")
				return
			}
			err = h.financeService.DeleteExpense(ctx, userID, expenseID)
			if err != nil {
				log.Printf("Failed to delete expense: %v", err)
				h.answerCallback(query.ID, "❌ Ошибка при удалении")
				return
			}
			fmt.Sprintf("expense_id: %d", expenseID)
			h.answerCallback(query.ID, "✅ Расход удален")
			h.handleShowExpenses(&tgbotapi.Message{From: &tgbotapi.User{ID: userID}, Chat: &tgbotapi.Chat{ID: chatID}})

		case "goal":
			goalID, err := strconv.ParseInt(resourceID, 10, 64)
			if err != nil {
				h.answerCallback(query.ID, "❌ Ошибка")
				return
			}
			err = h.financeService.DeleteGoal(ctx, userID, goalID)
			if err != nil {
				log.Printf("Failed to delete goal: %v", err)
				h.answerCallback(query.ID, "❌ Ошибка при удалении")
				return
			}
			fmt.Sprintf("goal_id: %d", goalID)
			h.answerCallback(query.ID, "✅ Цель удалена")
			h.handleShowGoals(&tgbotapi.Message{From: &tgbotapi.User{ID: userID}, Chat: &tgbotapi.Chat{ID: chatID}})
		}
		return

	case "select_goal":
		if params == "" {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		goalID, err := strconv.ParseInt(params, 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}

		h.answerCallback(query.ID, "✅")
		h.showGoalDetails(userID, chatID, goalID)
		return

	case "contrib":
		if params == "" {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}
		h.stateManager.SetTempData(userID, "contribute_goal_id", params)
		h.stateManager.SetState(userID, state.StateAddingContribution)
		h.answerCallback(query.ID, "✅ Введите сумму")
		h.sendMessage(chatID, "Введите сумму для добавления к цели:")
		return

	case "contribute":
		if params == "" {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}
		h.stateManager.SetTempData(userID, "contribute_goal_id", params)
		h.stateManager.SetState(userID, state.StateAddingContribution)
		h.answerCallback(query.ID, "✅ Введите сумму")
		h.sendMessage(chatID, "Введите сумму для добавления к цели:")
		return

	case "withdraw":
		if params == "" {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}
		goalID, err := strconv.ParseInt(params, 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}
		goal, err := h.financeService.GetUserGoalByID(ctx, userID, goalID)
		if err != nil {
			log.Printf("Failed to get goal: %v", err)
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}
		if goal.CurrentAmount == 0 {
			h.answerCallback(query.ID, "ℹ️ На цели нет средств")
			return
		}
		h.stateManager.SetTempData(userID, "withdraw_goal_id", params)
		h.stateManager.SetState(userID, state.StateWithdrawingFromGoal)
		h.answerCallback(query.ID, "✅ Введите сумму для вычета")
		h.sendMessage(chatID, fmt.Sprintf(
			"💸 Вычитание из цели: %s\nТекущая сумма: %d₽\n\nВведите сумму для вычета:",
			goal.GoalName, goal.CurrentAmount,
		))
		return

	case "changepriority":
		if params == "" {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		goalID, err := strconv.ParseInt(params, 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}

		h.answerCallback(query.ID, "✅")
		h.handleChangePriority(userID, chatID, goalID)
		return

	default:
		log.Printf("⚠️ DEBUG: Неизвестный callback: '%s', action: '%s', params: '%s'", callbackData, action, params)
		h.answerCallback(query.ID, fmt.Sprintf("❓ Неизвестное действие: %s", callbackData))
	}
}

func (h *BotHandler) handlePaydayCallbacks(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	callbackData := query.Data
	ctx := context.Background()

	parts := strings.Split(callbackData, "_")
	if len(parts) < 2 {
		h.answerCallback(query.ID, "❌ Ошибка формата")
		return
	}

	action := parts[1]

	switch action {
	case "goal":
		if len(parts) < 4 {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		incomeID, _ := strconv.ParseInt(parts[2], 10, 64)
		goalID, _ := strconv.ParseInt(parts[3], 10, 64)

		h.showGoalDetailsV2WithBack(userID, chatID, goalID, fmt.Sprintf("payday_back_%d", incomeID), "⬅️ Назад к получке")
		h.answerCallback(query.ID, "✅")

	case "add":
		if len(parts) < 4 {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		goalID, _ := strconv.ParseInt(parts[3], 10, 64)
		incomeID, _ := strconv.ParseInt(parts[2], 10, 64)

		h.stateManager.SetTempData(userID, "payday_contributing_goal_id", fmt.Sprintf("%d", goalID))
		h.stateManager.SetTempData(userID, "payday_contributing_income_id", fmt.Sprintf("%d", incomeID))
		h.stateManager.SetState(userID, state.StatePaydayEnteringAmount)

		h.sendMessage(chatID, "Введите сумму для отложения:")
		h.answerCallback(query.ID, "✅ Введите сумму")

	case "back":
		if len(parts) < 3 {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}

		incomeID, _ := strconv.ParseInt(parts[2], 10, 64)

		income, err := h.financeService.GetUserIncomeByID(ctx, userID, incomeID)
		if err != nil {
			log.Printf("Failed to get income by ID: %v", err)
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}

		if income == nil {
			h.answerCallback(query.ID, "❌ Доход не найден")
			return
		}

		h.showPaydayMenu(userID, chatID, income.ID, income.Name, income.Amount, ctx)
		h.answerCallback(query.ID, "✅")

	case "complete":
		// payday_complete_
		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(chatID, "😊 Взносы завершены! Спасибо!", h.mainMenu())
		h.answerCallback(query.ID, "✅ Готово")

	default:
		h.answerCallback(query.ID, "❓ Неизвестное действие")
	}
}

func (h *BotHandler) handleTestPaydayCallbacks(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	callbackData := query.Data
	ctx := context.Background()

	parts := strings.Split(callbackData, "_")
	if len(parts) < 3 {
		h.answerCallback(query.ID, "❌ Ошибка формата теста")
		return
	}

	action := parts[2]

	switch action {
	case "goal":
		if len(parts) < 5 {
			h.answerCallback(query.ID, "❌ Ошибка формата теста")
			return
		}

		incomeID, _ := strconv.ParseInt(parts[3], 10, 64)
		goalID, _ := strconv.ParseInt(parts[4], 10, 64)

		// детальную информацию цели с кнопкой назад к тестовому меню
		h.showTestGoalDetailsV2WithBack(userID, chatID, goalID, incomeID)
		h.answerCallback(query.ID, "✅ (Тест)")

	case "back":
		if len(parts) < 4 {
			h.answerCallback(query.ID, "❌ Ошибка формата теста")
			return
		}

		incomeID, _ := strconv.ParseInt(parts[3], 10, 64)

		err := h.financeService.TestPaydayNotification(h.bot, ctx, userID, incomeID)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка теста")
		}
		h.answerCallback(query.ID, "✅ (Тест)")

	case "add":
		if len(parts) < 5 {
			h.answerCallback(query.ID, "❌ Ошибка формата теста")
			return
		}

		incomeID, _ := strconv.ParseInt(parts[3], 10, 64)
		goalID, _ := strconv.ParseInt(parts[4], 10, 64)

		h.stateManager.SetTempData(userID, "payday_contributing_goal_id", fmt.Sprintf("%d", goalID))
		h.stateManager.SetTempData(userID, "payday_contributing_income_id", fmt.Sprintf("%d", incomeID))
		h.stateManager.SetState(userID, state.StatePaydayEnteringAmount)

		h.sendMessage(chatID, "🧪 Введите сумму для тестового вклада:")
		h.answerCallback(query.ID, "✅ Тест: Введите сумму")

	case "complete":
		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(chatID, "🧪 Тест завершен! Уведомления работают правильно.", h.mainMenu())
		h.answerCallback(query.ID, "✅ Тест завершен")

	default:
		h.answerCallback(query.ID, "❓ Неизвестное действие теста")
	}
}

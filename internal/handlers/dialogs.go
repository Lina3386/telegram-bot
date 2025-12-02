package handlers

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"strconv"
	"time"
)

func (h *BotHandler) HandleTextMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	text := message.Text
	ctx := context.Background()

	currentState := h.stateManager.GetState(userID)

	switch text {
	case "➕ Добавить доход":
		h.stateManager.SetState(userID, state.StateAddingIncome)
		h.sendMessage(chatID, "Введите название дохода (например: Зарплата, Пособие):")
		return

	case "📊 Мои доходы":
		h.handleShowIncomes(message)
		return

	case "💰 Мои расходы":
		h.handleShowExpenses(message)
		return

	case "🎯 Цели":
		h.handleShowGoals(message)
		return

	case "📈 Статистика":
		h.handleShowStats(message)
		return

	case "⬅️ Назад":
		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(chatID, "Вернулись в главное меню", h.mainMenu())
		return
	}

	switch currentState {
	case state.StateAddingIncome:
		if err := ValidateExpenseName(text); err != nil {
			h.sendMessage(chatID, err.Error())
			return
		}
		h.stateManager.SetTempData(userID, "income_name", text)
		h.stateManager.SetState(userID, state.StateAddingIncomeAmount)
		h.sendMessage(chatID, "Введите размер дохода (в рублях):")

	case state.StateAddingIncomeAmount:
		amount, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			h.sendMessage(chatID, "Введите число (без букв и символов)")
			return
		}
		if err := ValidateAmount(amount); err != nil {
			h.sendMessage(chatID, err.Error())
			return
		}
		if amount <= 0 {
			h.sendMessage(chatID, "Сумма должна быть положительной")
			return
		}
		h.stateManager.SetTempData(userID, "income_amount", text)
		h.stateManager.SetState(userID, state.StateAddingIncomeDay)
		h.sendMessage(chatID, "Введите день месяца для получения дохода (1-31):")

	case state.StateAddingIncomeDay:
		day, err := strconv.Atoi(text)
		if err != nil || day < 1 || day > 31 {
			h.sendMessage(chatID, "❌ Введите число от 1 до 31")
			return
		}

		// Сохраняем доход в БД
		incomeName := h.stateManager.GetTempData(userID, "income_name")
		incomeAmount, _ := strconv.ParseInt(h.stateManager.GetTempData(userID, "income_amount"), 10, 64)

		nextPayDate := time.Now().AddDate(0, 0, day-time.Now().Day())
		if nextPayDate.Before(time.Now()) {
			nextPayDate = nextPayDate.AddDate(0, 1, 0)
		}

		_, err = h.financeService.CreateIncome(ctx, userID, incomeName, incomeAmount, day, nextPayDate)
		if err != nil {
			log.Printf("Failed to create income: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при сохранении дохода")
			return
		}

		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("✅ Доход добавлен:\n%s: %d₽ (дата: %d число)", incomeName, incomeAmount, day),
			h.mainMenu(),
		)

	case state.StateAddingExpense:
		h.stateManager.SetTempData(userID, "expense_name", text)
		h.stateManager.SetState(userID, state.StateAddingExpenseAmount)
		h.sendMessage(chatID, "Введите размер расхода (в рублях):")

	case state.StateAddingExpenseAmount:
		amount, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			h.sendMessage(chatID, "Введите число (без букв и символов)")
			return
		}
		if err := ValidateAmount(amount); err != nil {
			h.sendMessage(chatID, err.Error())
			return
		}
		if amount <= 0 {
			h.sendMessage(chatID, "Сумма должна быть положительной")
			return
		}

		expenseName := h.stateManager.GetTempData(userID, "expense_name")

		if err := ValidateExpenseName(expenseName); err != nil {
			h.sendMessage(chatID, err.Error())
			return
		}

		_, err = h.financeService.CreateExpense(ctx, userID, expenseName, amount)
		if err != nil {
			log.Printf("Failed to create expense: %v", err)
			h.sendMessage(chatID, "Ошибка при сохранении расхода")
			return
		}

		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("Расход добавлен:\\n💳 %s: %d₽", expenseName, amount),
			h.mainMenu(),
		)

	case state.StateCreatingGoal:
		h.stateManager.SetTempData(userID, "goal_name", text)
		h.stateManager.SetState(userID, state.StateCreatingGoalTarget)
		h.sendMessage(chatID, "Введите целевую сумму (в рублях):")

	case state.StateCreatingGoalTarget:
		targetAmount, err := strconv.ParseInt(text, 10, 64)
		if err != nil || targetAmount <= 0 {
			h.sendMessage(chatID, "❌ Введите корректное число")
			return
		}

		goalName := h.stateManager.GetTempData(userID, "goal_name")

		// Создаем цель
		goal, err := h.financeService.CreateGoal(ctx, userID, goalName, targetAmount)
		if err != nil {
			log.Printf("Failed to create goal: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при создании цели")
			return
		}

		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("✅ Цель создана:\n%s\nЦель: %d₽\nМесячный взнос: %d₽\nДата достижения: %s",
				goalName, targetAmount, goal.MonthlyContrib, goal.TargetDate.Format("02.01.2006")),
			h.mainMenu(),
		)

	default:
		// Предлагаем меню если нет активного состояния
		if currentState == state.StateIdle {
			h.sendMessageWithKeyboard(chatID, "Используйте меню ниже:", h.mainMenu())
		}
	}
}

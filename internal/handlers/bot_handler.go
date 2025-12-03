package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Lina3386/telegram-bot/internal/client"
	"github.com/Lina3386/telegram-bot/internal/services"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotHandler struct {
	bot            *tgbotapi.BotAPI
	financeService *services.FinanceService
	authClient     *client.AuthClient
	chatClient     *client.ChatClient
	stateManager   *state.StateManager
}

func NewBotHandler(
	bot *tgbotapi.BotAPI,
	financeService *services.FinanceService,
	authClient *client.AuthClient,
	chatClient *client.ChatClient,
	stateManager *state.StateManager,
) *BotHandler {
	return &BotHandler{
		bot:            bot,
		financeService: financeService,
		authClient:     authClient,
		chatClient:     chatClient,
		stateManager:   stateManager,
	}
}

func (h *BotHandler) HandleStart(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	}
	chatID := message.Chat.ID

	ctx := context.Background()
	log.Printf("User %d (%s) started the bot", userID, username)

	token, err := h.authClient.RegisterTelegramUser(ctx, userID, username)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		h.sendMessage(chatID, "Ошибка регистрации. Попробуйте позже.")
		return
	}
	_ = h.chatClient.LogFinancialOperation(ctx, userID, "USER_REGISTERED", fmt.Sprintf("User %s registered", username))
	_, err = h.financeService.CreateUser(ctx, userID, username, token)
	if err != nil {
		log.Printf("Failed to create user in DB: %v", err)
		h.sendMessage(chatID, "Ошибка при сохранении данных.")
		return
	}
	h.stateManager.ClearState(userID)

	msg := fmt.Sprintf("👋 Добро пожаловать, %s!\n\n"+
		"Я помогу вам управлять финансами.\n\n"+
		"Выберите действие:",
		username,
	)

	h.sendMessageWithKeyboard(chatID, msg, h.mainMenu())
}

func (h *BotHandler) HandleHelp(message *tgbotapi.Message) {
	helpText := `📖 Справка по командам:

/start - Начать работу
/help - Показать эту справку
/cancel - Отменить текущее действие

📌 Как использовать:
1️⃣ Нажмите ➕ чтобы добавить доход
2️⃣ Нажмите 💰 чтобы добавить расход
3️⃣ Нажмите 🎯 чтобы создать цель
4️⃣ Нажмите 📈 чтобы увидеть статистику

💡 Совет: Все действия можно отменить командой /cancel`

	h.sendMessage(message.Chat.ID, helpText)
}

func (h *BotHandler) HandleCancel(message *tgbotapi.Message) {
	userID := message.From.ID
	currentState := h.stateManager.GetState(userID)

	if currentState == state.StateIdle {
		h.sendMessage(message.Chat.ID, "ℹ️ Нет активного действия для отмены")
		return
	}

	h.stateManager.ClearState(userID)
	h.sendMessageWithKeyboard(message.Chat.ID, "❌ Действие отменено. Вернулись в главное меню", h.mainMenu())
}

func (h *BotHandler) HandleUnknownCommand(message *tgbotapi.Message) {
	h.sendMessage(message.Chat.ID, "❓ Неизвестная команда.\n\nИспользуйте /help для справки")
}

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
		h.stateManager.SetTempData(userID, "income_name", text)
		h.stateManager.SetState(userID, state.StateAddingIncomeAmount)
		h.sendMessage(chatID, "Введите размер дохода (в рублях):")

	case state.StateAddingIncomeAmount:
		amount, err := strconv.ParseInt(text, 10, 64)
		if err != nil || amount <= 0 {
			h.sendMessage(chatID, "❌ Введите корректное число")
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

		_ = h.chatClient.LogFinancialOperation(ctx, userID, "INCOME_ADDED", fmt.Sprintf("%s: %d₽ (day %d)", incomeName, incomeAmount, day))

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
		if err != nil || amount <= 0 {
			h.sendMessage(chatID, "❌ Введите корректное число")
			return
		}

		expenseName := h.stateManager.GetTempData(userID, "expense_name")

		_, err = h.financeService.CreateExpense(ctx, userID, expenseName, amount)
		if err != nil {
			log.Printf("Failed to create expense: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при сохранении расхода")
			return
		}

		_ = h.chatClient.LogFinancialOperation(ctx, userID, "EXPENSE_ADDED", fmt.Sprintf("%s: %d₽", expenseName, amount))

		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("✅ Расход добавлен:\n%s: %d₽", expenseName, amount),
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

		h.stateManager.SetTempData(userID, "goal_target", text)
		h.stateManager.SetState(userID, state.StateCreatingGoalPriority)
		h.sendMessage(chatID, "Выберите приоритет цели:\n\n1️⃣ Высший (1)\n2️⃣ Средний (2)\n3️⃣ Низкий (3)\n\nВведите число от 1 до 3:")

	case state.StateCreatingGoalPriority:
		priority, err := strconv.Atoi(text)
		if err != nil || priority < 1 || priority > 3 {
			h.sendMessage(chatID, "❌ Введите число от 1 до 3")
			return
		}

		goalName := h.stateManager.GetTempData(userID, "goal_name")
		targetAmount, _ := strconv.ParseInt(h.stateManager.GetTempData(userID, "goal_target"), 10, 64)

		goal, err := h.financeService.CreateGoal(ctx, userID, goalName, targetAmount, priority)
		if err != nil {
			log.Printf("Failed to create goal: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при создании цели")
			return
		}

		priorityText := []string{"", "Высший", "Средний", "Низкий"}[priority]
		_ = h.chatClient.LogFinancialOperation(ctx, userID, "GOAL_CREATED", fmt.Sprintf("%s: %d₽ (priority: %s)", goalName, targetAmount, priorityText))
		timeToGoal := h.calculateTimeToGoal(targetAmount, goal.MonthlyContrib, 0)

		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("✅ Цель создана:\n%s\nЦель: %d₽\nМесячный взнос: %d₽\nПриоритет: %s (%d)\nВремя до цели: %s\nДата достижения: %s",
				goalName, targetAmount, goal.MonthlyContrib, priorityText, priority, timeToGoal, goal.TargetDate.Format("02.01.2006")),
			h.mainMenu(),
		)

	case state.StateWithdrawingFromGoal:
		amount, err := strconv.ParseInt(text, 10, 64)
		if err != nil || amount <= 0 {
			h.sendMessage(chatID, "❌ Введите корректное число")
			return
		}

		goalIDStr := h.stateManager.GetTempData(userID, "withdraw_goal_id")
		goalID, err := strconv.ParseInt(goalIDStr, 10, 64)
		if err != nil {
			h.sendMessage(chatID, "❌ Ошибка")
			h.stateManager.ClearState(userID)
			return
		}

		goal, err := h.financeService.WithdrawFromGoal(ctx, goalID, amount)
		if err != nil {
			log.Printf("Failed to withdraw from goal: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при вычитании")
			h.stateManager.ClearState(userID)
			return
		}

		_ = h.chatClient.LogFinancialOperation(ctx, userID, "GOAL_WITHDRAWAL", fmt.Sprintf("%s: -%d₽ (remaining: %d₽)", goal.GoalName, amount, goal.CurrentAmount))

		progress := int64(0)
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount * 100) / goal.TargetAmount
		}

		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("✅ Вычтено %d₽\n\n🎯 %s\nОсталось: %d₽ / %d₽ (%d%%)",
				amount, goal.GoalName, goal.CurrentAmount, goal.TargetAmount, progress),
			h.mainMenu(),
		)

	default:
		if currentState == state.StateIdle {
			h.sendMessageWithKeyboard(chatID, "Используйте меню ниже:", h.mainMenu())
		}
	}
}

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
	priorityNames := map[int]string{1: "🔴 Высший", 2: "🟡 Средний", 3: "🟢 Низкий"}
	for _, goal := range goals {
		progress := int64(0)
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount * 100) / goal.TargetAmount
		}
		priorityText := priorityNames[goal.Priority]
		if priorityText == "" {
			priorityText = "🟡 Средний"
		}
		timeToGoal := h.calculateTimeToGoal(goal.TargetAmount, goal.MonthlyContrib, goal.CurrentAmount)
		statusText := goal.Status
		if statusText == "active" {
			statusText = "Активна"
		} else if statusText == "completed" {
			statusText = "Завершена ✅"
		}
		text += fmt.Sprintf(
			"%s %s\nЦель: %d₽ | Собрано: %d₽ (%d%%)\nМесячный взнос: %d₽\nВремя до цели: %s\nДата: %s | Статус: %s\n\n",
			priorityText, goal.GoalName, goal.TargetAmount, goal.CurrentAmount, progress,
			goal.MonthlyContrib, timeToGoal, goal.TargetDate.Format("02.01.2006"), statusText,
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

	goals, err := h.financeService.GetUserActiveGoalsByTelegramID(ctx, userID)
	if err != nil {
		log.Printf("Failed to get goals: %v", err)
	}

	text := fmt.Sprintf(
		"📈 Ваша финансовая статистика:\n\n"+
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

func (h *BotHandler) HandleCallback(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	callbackData := query.Data

	log.Printf("Callback from user %d: %s", userID, callbackData)
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

	case "add_contribution":
		if len(parts) < 3 {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}
		goalID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}
		amount, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}

		ctx := context.Background()
		goal, err := h.financeService.ContributeToGoal(ctx, goalID, amount)
		if err != nil {
			log.Printf("Failed to contribute to goal: %v", err)
			h.answerCallback(query.ID, "❌ Ошибка при добавлении")
			return
		}

		progress := int64(0)
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount * 100) / goal.TargetAmount
		}

		statusText := "✅ Добавлено!"
		if goal.Status == "completed" {
			statusText = "🎉 Цель достигнута!"
		}

		ctx = context.Background()
		_ = h.chatClient.LogFinancialOperation(ctx, userID, "GOAL_CONTRIBUTION", fmt.Sprintf("%s: +%d₽ (total: %d₽)", goal.GoalName, amount, goal.CurrentAmount))

		h.answerCallback(query.ID, statusText)
		h.sendMessage(chatID, fmt.Sprintf(
			"%s\n\n🎯 %s\nСобрано: %d₽ / %d₽ (%d%%)",
			statusText, goal.GoalName, goal.CurrentAmount, goal.TargetAmount, progress,
		))

	case "withdraw":
		if len(parts) < 2 {
			h.answerCallback(query.ID, "❌ Ошибка формата")
			return
		}
		goalID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			h.answerCallback(query.ID, "❌ Ошибка")
			return
		}

		ctx := context.Background()
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

		h.stateManager.SetTempData(userID, "withdraw_goal_id", parts[1])
		h.stateManager.SetState(userID, state.StateWithdrawingFromGoal)
		h.answerCallback(query.ID, "✅ Введите сумму для вычета")
		h.sendMessage(chatID, fmt.Sprintf(
			"💸 Вычитание из цели: %s\nТекущая сумма: %d₽\n\nВведите сумму для вычета:",
			goal.GoalName, goal.CurrentAmount,
		))

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

func (h *BotHandler) calculateTimeToGoal(targetAmount, monthlyContrib, currentAmount int64) string {
	remaining := targetAmount - currentAmount
	if remaining <= 0 {
		return "Цель достигнута! 🎉"
	}

	if monthlyContrib <= 0 {
		return "Недостаточно средств для накопления"
	}

	months := remaining / monthlyContrib
	if remaining%monthlyContrib > 0 {
		months++
	}

	years := months / 12
	months = months % 12
	days := (remaining % monthlyContrib) * 30 / monthlyContrib

	var parts []string
	if years > 0 {
		yearWord := "лет"
		if years == 1 {
			yearWord = "год"
		} else if years >= 2 && years <= 4 {
			yearWord = "года"
		}
		parts = append(parts, fmt.Sprintf("%d %s", years, yearWord))
	}
	if months > 0 {
		monthWord := "месяцев"
		if months == 1 {
			monthWord = "месяц"
		} else if months >= 2 && months <= 4 {
			monthWord = "месяца"
		}
		parts = append(parts, fmt.Sprintf("%d %s", months, monthWord))
	}
	if days > 0 && years == 0 {
		dayWord := "дней"
		if days == 1 {
			dayWord = "день"
		} else if days >= 2 && days <= 4 {
			dayWord = "дня"
		}
		parts = append(parts, fmt.Sprintf("%d %s", days, dayWord))
	}

	if len(parts) == 0 {
		return "Меньше месяца"
	}

	return strings.Join(parts, " ")
}

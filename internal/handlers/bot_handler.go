package handlers

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/models"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Lina3386/telegram-bot/internal/services"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const helpText = `📖 Справка по командам:

/start - Начать работу
/help - Показать эту справку
/cancel - Отменить текущее действие

📌 Как использовать:
1️⃣ Нажмите 💳 чтобы добавить доход
2️⃣ Нажмите 💰 чтобы добавить расход
3️⃣ Нажмите 🍀 чтобы создать цель
4️⃣ Нажмите 📈 чтобы увидеть статистику

💡 Совет: Все действия можно отменить командой /cancel`

type BotHandler struct {
	bot            *tgbotapi.BotAPI
	financeService *services.FinanceService
	authService    *services.AuthService
	stateManager   *state.StateManager
}

func NewBotHandler(
	bot *tgbotapi.BotAPI,
	financeService *services.FinanceService,
	authService *services.AuthService,
	stateManager *state.StateManager,
) *BotHandler {
	return &BotHandler{
		bot:            bot,
		financeService: financeService,
		authService:    authService,
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

	// существует ли пользователь
	existingUser, err := h.financeService.GetUserByTelegramID(ctx, userID)
	if err == nil && existingUser != nil {
		// уже существует - просто приветствие
		log.Printf("User %d already exists, welcome back!", userID)
		h.stateManager.ClearState(userID)
		msg := fmt.Sprintf("👋 С возвращением, %s!\n\n"+
			"Выберите действие:\n\n"+
			helpText,
			username,
		)
		h.sendMessageWithKeyboard(chatID, msg, h.mainMenu())
		return
	}

	token, err := h.authService.RegisterTelegramUser(ctx, userID, username)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		h.sendMessage(chatID, "Ошибка регистрации. Попробуйте позже.")
		return
	}

	fmt.Sprintf("User %s registered", username)

	_, err = h.financeService.CreateUser(ctx, userID, username, token)
	if err != nil {
		log.Printf("Failed to create user in DB: %v", err)
		existingUser, checkErr := h.financeService.GetUserByTelegramID(ctx, userID)
		if checkErr == nil && existingUser != nil {
			log.Printf("User already exists, continuing...")
			h.stateManager.ClearState(userID)
			msg := fmt.Sprintf("👋 Добро пожаловать, %s!\n\n"+
				"Я помогу вам управлять финансами.\n\n"+
				"Выберите действие:\n\n"+
				helpText,
				username,
			)
			h.sendMessageWithKeyboard(chatID, msg, h.mainMenu())
			return
		}
		h.sendMessage(chatID, "Ошибка при сохранении данных.")
		return
	}

	h.stateManager.ClearState(userID)

	msg := fmt.Sprintf("👋 Добро пожаловать, %s!\n\n"+
		"Я помогу вам управлять финансами.\n\n"+
		"Выберите действие:\n\n"+
		helpText,
		username,
	)

	h.sendMessageWithKeyboard(chatID, msg, h.mainMenu())
}

func (h *BotHandler) HandleHelp(message *tgbotapi.Message) {
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
	if strings.HasPrefix(message.Text, "/testpayday") {
		h.handleTestPaydayCommand(message)
		return
	}

	h.sendMessage(message.Chat.ID, "❓ Неизвестная команда.\n\nИспользуйте /help для справки")
}

func (h *BotHandler) handleTestPaydayCommand(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	args := strings.Fields(message.Text)
	if len(args) < 2 {
		h.sendMessage(chatID, "❌ Использование: /testpayday [income_id]\n\nСначала посмотрите список своих доходов")
		return
	}

	incomeIDStr := args[1]
	incomeID, err := strconv.ParseInt(incomeIDStr, 10, 64)
	if err != nil {
		h.sendMessage(chatID, "❌ ID дохода должен быть числом")
		return
	}

	err = h.financeService.TestPaydayNotification(h.bot, ctx, userID, incomeID)
	if err != nil {
		h.sendMessage(chatID, fmt.Sprintf("❌ Ошибка: %v", err))
		return
	}

	h.sendMessage(chatID, "✅ Тестовое уведомление отправлено!")
}

func (h *BotHandler) HandleTextMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	text := message.Text
	ctx := context.Background()

	currentState := h.stateManager.GetState(userID)

	switch text {
	case "💳 Мои доходы", "мои доходы", "доходы":
		h.handleShowIncomes(message)
		return

	case "💰 Мои расходы", "мои расходы", "расходы":
		h.handleShowExpenses(message)
		return

	case "🍀 Цели", "цели", "цель":
		h.handleShowGoals(message)
		return

	case "📈 Статистика", "📊 Статистика", "статистика", "стата":
		h.handleShowStats(message)
		return

	case "✅ Готово", "готово":
		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(chatID, "Операция завершена!", h.mainMenu())
		return

	case "⬅️ Назад", "назад":
		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(chatID, "Вернулись в главное меню", h.mainMenu())
		return
	}

	switch currentState {
	case state.StateChangingGoalPriority:
		h.handlePriorityInput(message)
		return

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
		h.stateManager.SetState(userID, state.StateAddingIncomeFrequency)
		h.sendMessage(chatID, "Выберите частоту получения дохода:\n\n1️⃣ Ежемесячно (monthly)\n2️⃣ Еженедельно (weekly)\n3️⃣ Через неделю (biweekly)\n\nВведите число от 1 до 3:")

	case state.StateAddingIncomeFrequency:
		freq, _ := strconv.Atoi(text)
		var frequency string
		var prompt string
		switch freq {
		case 1:
			frequency = "monthly"
			prompt = "Введите день месяца для получения дохода (1-31):"
		case 2:
			frequency = "weekly"
			prompt = "Введите день недели для получения дохода (0=воскресенье, 1=понедельник, ..., 6=суббота):"
		case 3:
			frequency = "biweekly"
			prompt = "Введите день недели для получения дохода (0=воскресенье, 1=понедельник, ..., 6=суббота):"
		default:
			h.sendMessage(chatID, "❌ Введите число от 1 до 3")
			return
		}
		h.stateManager.SetTempData(userID, "income_frequency", frequency)
		h.stateManager.SetState(userID, state.StateAddingIncomeDay)
		h.sendMessage(chatID, prompt)

	case state.StateAddingIncomeDay:
		recurringDay, err := strconv.Atoi(text)
		frequency := h.stateManager.GetTempData(userID, "income_frequency")
		if frequency == "" {
			frequency = "monthly"
		}

		if frequency == "monthly" && (err != nil || recurringDay < 1 || recurringDay > 31) {
			h.sendMessage(chatID, "❌ Введите число от 1 до 31")
			return
		}
		if (frequency == "weekly" || frequency == "biweekly") && (err != nil || recurringDay < 0 || recurringDay > 6) {
			h.sendMessage(chatID, "❌ Введите число от 0 до 6 (день недели)")
			return
		}

		h.stateManager.SetTempData(userID, "income_recurring_day", text)
		h.stateManager.SetState(userID, state.StateAddingIncomeHour)
		h.sendMessage(chatID, "В каком часу получать уведомления? (0-23, например 9 для 9:00, 18 для 18:00)\n\nПо умолчанию: 18:00")

	case state.StateAddingIncomeHour:
		notificationHour, err := strconv.Atoi(text)
		if err != nil || notificationHour < 0 || notificationHour > 23 {
			h.sendMessage(chatID, "❌ Введите число от 0 до 23")
			return
		}

		incomeName := h.stateManager.GetTempData(userID, "income_name")
		incomeAmount, _ := strconv.ParseInt(h.stateManager.GetTempData(userID, "income_amount"), 10, 64)
		frequency := h.stateManager.GetTempData(userID, "income_frequency")
		recurringDay, _ := strconv.Atoi(h.stateManager.GetTempData(userID, "income_recurring_day"))

		nextPayDate := h.calculateNextPayDate(frequency, recurringDay)

		_, err = h.financeService.CreateIncomeWithFrequencyAndHour(ctx, userID, incomeName, incomeAmount, frequency, recurringDay, notificationHour, nextPayDate)
		if err != nil {
			log.Printf("Failed to create income: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при сохранении дохода")
			return
		}

		fmt.Sprintf("%s: %d₽ (%s day %d, notify at %d:00)", incomeName, incomeAmount, frequency, recurringDay, notificationHour)

		h.stateManager.ClearState(userID)

		dayDesc := fmt.Sprintf("число %d", recurringDay)
		if frequency == "weekly" || frequency == "biweekly" {
			weeks := map[int]string{0: "воскресенье", 1: "понедельник", 2: "вторник", 3: "среда", 4: "четверг", 5: "пятница", 6: "суббота"}
			dayDesc = weeks[recurringDay]
		}

		freqDesc := map[string]string{
			"monthly":  "ежемесячно",
			"weekly":   "еженедельно",
			"biweekly": "через неделю",
		}[frequency]

		h.sendMessageWithKeyboard(
			chatID,
			fmt.Sprintf("✅ Доход добавлен:\n%s: %d₽ (%s, %s)\n🔔 Уведомления в %d:00", incomeName, incomeAmount, freqDesc, dayDesc, notificationHour),
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

		fmt.Sprintf("%s: %d₽", expenseName, amount)

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
			h.sendMessage(chatID, "❌ Введите корректную сумму")
			return
		}
		h.stateManager.SetTempData(userID, "goal:target", text)

		allGoals, err := h.financeService.GetUserGoals(ctx, userID)
		if err != nil {
			h.sendMessage(chatID, "❌ Ошибка при получении списка целей")
			h.stateManager.ClearState(userID)
			return
		}

		maxPriority := 0
		for _, g := range allGoals {
			if g.Status == "active" && g.Priority > maxPriority {
				maxPriority = int(g.Priority)
			}
		}
		newPriority := maxPriority + 1

		goalName := h.stateManager.GetTempData(userID, "goal_name")
		goal, err := h.financeService.CreateGoal(ctx, userID, goalName, targetAmount, newPriority)
		if err != nil {
			log.Printf("Failed to create goal: %v", err)
			h.sendMessage(chatID, "❌ Ошибка создания цели")
			return
		}

		var priorityText string
		if newPriority == 1 {
			priorityText = "Наивысший"
		} else if newPriority == 2 {
			priorityText = "Высокий"
		} else if newPriority == 3 {
			priorityText = "Низкий"
		} else {
			priorityText = fmt.Sprintf("Приоритет %d", newPriority)
		}

		fmt.Sprintf("%s (цель: %d, приоритет: %s)", goalName, targetAmount, priorityText)

		timeToGoal := h.calculateTimeToGoal(targetAmount, goal.MonthlyContrib, 0)
		h.stateManager.ClearState(userID)
		h.sendMessageWithKeyboard(chatID, fmt.Sprintf("✅ Цель создана:\n📌 %s\n💰 Сумма: %d₽\n📅 Ежемесячно: %d₽\n⚡ Приоритет: %s (%d)\n⏱ Время до цели: %s\n📆 Дата достижения: %s", goalName, targetAmount, goal.MonthlyContrib, priorityText, newPriority, timeToGoal, goal.TargetDate.Format("02.01.2006")), h.mainMenu())
		return

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

		fmt.Sprintf("%s: -%d₽ (remaining: %d₽)", goal.GoalName, amount, goal.CurrentAmount)

		progress := int64(0)
		if goal.TargetAmount > 0 {
			progress = (goal.CurrentAmount * 100) / goal.TargetAmount
		}

		h.stateManager.ClearState(userID)

		// Кнопка вернуться к цели
		backToGoalBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Вернуться к цели", fmt.Sprintf("select_goal_%d", goal.ID))

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"✅ Вычтено %d₽\n\n🎯 %s\nОсталось: %d₽ / %d₽ (%d%%)",
			amount, goal.GoalName, goal.CurrentAmount, goal.TargetAmount, progress))

		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{backToGoalBtn})
		h.bot.Send(msg)
	case state.StateAddingContribution:
		amount, err := strconv.ParseInt(text, 10, 64)
		if err != nil || amount <= 0 {
			h.sendMessage(chatID, "❌ Введите корректное число")
			return
		}

		goalIDStr := h.stateManager.GetTempData(userID, "contribute_goal_id")
		goalID, _ := strconv.ParseInt(goalIDStr, 10, 64)

		goal, err := h.financeService.ContributeToGoal(ctx, goalID, amount)
		if err != nil {
			log.Printf("Failed to contribute to goal: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при добавлении")
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

		fmt.Sprintf("%s: +%d₽ (total: %d₽)", goal.GoalName, amount, goal.CurrentAmount)

		h.stateManager.ClearState(userID)

		backToGoalBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 Вернуться к цели", fmt.Sprintf("select_goal_%d", goal.ID))

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
			"%s\n\n🎯 %s\nСобрано: %d₽ / %d₽ (%d%%)",
			statusText, goal.GoalName, goal.CurrentAmount, goal.TargetAmount, progress))

		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{backToGoalBtn})
		h.bot.Send(msg)

	case state.StatePaydayEnteringAmount:
		h.handlePaydayAmountInput(message)
		return

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

	text := "💳 Ваши доходы:\n\n"
	var inlineButtons [][]tgbotapi.InlineKeyboardButton

	if len(incomes) > 0 {
		totalIncome, err := h.financeService.CalculateTotalIncome(ctx, userID)
		if err != nil {
			log.Printf("Failed to calculate total income: %v", err)
			totalIncome = 0
		}

		for _, income := range incomes {
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

			text += fmt.Sprintf("%d\n💰 %s: %d₽ (%s, %s)\n\n", income.ID, income.Name, income.Amount, freqText, dayDesc)

			button := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🗑️ Удалить %d", income.ID), fmt.Sprintf("delete_income_%d", income.ID))
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
			"🎯 День получки: %s\n"+
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

func (h *BotHandler) mainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💳 Мои доходы"),
			tgbotapi.NewKeyboardButton("💰 Мои расходы"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🍀 Цели"),
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
		h.sendMessage(chatID, "❌ Ошибка при возврате к меню получки")
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
		h.sendMessage(chatID, "❌ Не найден доход для меню получки")
		return
	}

	h.stateManager.ClearState(userID)
	h.showPaydayMenu(userID, chatID, incomeID, incomeName, incomeAmount, ctx)
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

// изменение приоритета
func (h *BotHandler) handleChangePriority(userID int64, chatID int64, goalID int64) {
	ctx := context.Background()

	goals, err := h.financeService.GetUserGoals(ctx, userID)
	if err != nil {
		h.sendMessage(chatID, "❌ Ошибка при загрузке целей")
		return
	}

	activeGoalsCount := 0
	for _, goal := range goals {
		if goal.Status == "active" {
			activeGoalsCount++
		}
	}

	if activeGoalsCount <= 1 {
		h.sendMessage(chatID, "ℹ️ Невозможно изменить приоритет: требуется минимум 2 активные цели")
		return
	}

	h.stateManager.SetTempData(userID, "change_priority_goal_id", fmt.Sprintf("%d", goalID))
	h.stateManager.SetState(userID, state.StateChangingGoalPriority)

	text := fmt.Sprintf(
		"🔀 <b>Изменение приоритета</b>\n\n"+
			"Выберите новый приоритет от 1 до %d:\n\n"+
			"1️⃣ = Наивысший приоритет\n"+
			"%d = Низший приоритет\n\n"+
			"Введите число:",
		activeGoalsCount,
		activeGoalsCount,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	h.bot.Send(msg)
}

func (h *BotHandler) handlePriorityInput(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	ctx := context.Background()

	newPriority, err := strconv.Atoi(message.Text)
	if err != nil {
		h.sendMessage(chatID, "❌ Введите корректное число")
		return
	}

	goalIDStr := h.stateManager.GetTempData(userID, "change_priority_goal_id")
	goalID, _ := strconv.ParseInt(goalIDStr, 10, 64)

	goals, err := h.financeService.GetUserGoals(ctx, userID)
	if err != nil {
		h.sendMessage(chatID, "❌ Ошибка при загрузке целей")
		h.stateManager.ClearState(userID)
		return
	}

	activeGoalsCount := 0
	for _, goal := range goals {
		if goal.Status == "active" {
			activeGoalsCount++
		}
	}

	if newPriority < 1 || newPriority > activeGoalsCount {
		h.sendMessage(chatID, fmt.Sprintf("❌ Введите число от 1 до %d", activeGoalsCount))
		return
	}

	err = h.financeService.SwapGoalPriorities(ctx, userID, goalID, newPriority)
	if err != nil {
		log.Printf("Failed to swap priorities: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при изменении приоритета")
		h.stateManager.ClearState(userID)
		return
	}

	_, err = h.financeService.DistributeFundsToGoals(ctx, userID)
	if err != nil {
		log.Printf("Failed to redistribute funds: %v", err)
	}

	h.stateManager.ClearState(userID)

	h.sendMessage(chatID, fmt.Sprintf("✅ Приоритет изменен на %d\n\nБюджет пересчитан в соответствии с новыми приоритетами", newPriority))

	time.Sleep(500 * time.Millisecond)
	h.showGoalDetailsV2(userID, chatID, goalID)
}

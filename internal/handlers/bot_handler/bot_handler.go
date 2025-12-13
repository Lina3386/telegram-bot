package bot_handler

import (
	"context"
	"fmt"
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

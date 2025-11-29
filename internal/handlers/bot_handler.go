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
	authClient     *client.AuthClient
	chatClient     *client.ChatClient
	financeService *services.FinanceService
	stateManager   *state.StateManager
}

func NewBotHandler(
	bot *tgbotapi.BotAPI,
	authClient *client.AuthClient,
	chatClient *client.ChatClient,
	financeService *services.FinanceService,
	stateManager *state.StateManager,
) *BotHandler {
	return &BotHandler{
		bot:            bot,
		authClient:     authClient,
		chatClient:     chatClient,
		financeService: financeService,
		stateManager:   stateManager,
	}
}

// ============ ОБРАБОТЧИКИ КОМАНД ============

func (h *BotHandler) HandleStart(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName
	if username == "" {
		username = message.From.FirstName
	}
	chatID := message.Chat.ID

	ctx := context.Background()
	log.Printf("User %d (%s) started the bot", userID, username)

	// ✅ Регистрируем пользователя через Auth сервис
	token, err := h.authClient.RegisterTelegramUser(ctx, userID, username)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		h.sendMessage(chatID, "❌ Ошибка регистрации. Попробуйте позже.")
		return
	}

	// ✅ Создаем пользователя в БД
	_, err = h.financeService.CreateUser(ctx, userID, username, token)
	if err != nil {
		log.Printf("Failed to create user in DB: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при сохранении данных.")
		return
	}

	// ✅ Очищаем состояние
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

// ============ ОБРАБОТЧИК ТЕКСТОВЫХ СООБЩЕНИЙ ============

func (h *BotHandler) HandleTextMessage(message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	text := message.Text
	ctx := context.Background()

	currentState := h.stateManager.GetState(userID)

	// ✅ Обработка текстовых меню
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

	// ✅ Обработка состояний диалога
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

		// ✅ Сохраняем доход в БД
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
		if err != nil || amount <= 0 {
			h.sendMessage(chatID, "❌ Введите корректное число")
			return
		}

		expenseName := h.stateManager.GetTempData(userID, "expense_name")

		// ✅ Сохраняем расход в БД
		_, err = h.financeService.CreateExpense(ctx, userID, expenseName, amount)
		if err != nil {
			log.Printf("Failed to create expense: %v", err)
			h.sendMessage(chatID, "❌ Ошибка при сохранении расхода")
			return
		}

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

		goalName := h.stateManager.GetTempData(userID, "goal_name")

		// ✅ Создаем цель
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
		// ✅ Предлагаем меню если нет активного состояния
		if currentState == state.StateIdle {
			h.sendMessageWithKeyboard(chatID, "Используйте меню ниже:", h.mainMenu())
		}
	}
}

// ============ ОБРАБОТЧИКИ МЕНЮ ============

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

// ============ ОБРАБОТЧИК КНОПОК (CallbackQuery) ============

func (h *BotHandler) HandleCallback(query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	chatID := query.Message.Chat.ID
	callbackData := query.Data

	log.Printf("Callback from user %d: %s", userID, callbackData)

	// ✅ Разбираем callback данные
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

// ============ ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ============

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

// ✅ Исправлено: обработка ошибок при отправке
func (h *BotHandler) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := h.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send message to %d: %v", chatID, err)
		return err
	}
	return nil
}

// ✅ Исправлено: обработка ошибок при отправке с клавиатурой
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

// ✅ Обработчик ответа на callback
func (h *BotHandler) answerCallback(callbackQueryID, text string) {
	callback := tgbotapi.NewCallback(callbackQueryID, text)
	h.bot.Request(callback)
}

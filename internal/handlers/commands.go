package handlers

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

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
		h.sendMessage(chatID, "❌ Ошибка регистрации. Попробуйте позже.")
		return
	}

	_, err = h.financeService.CreateUser(ctx, userID, username, token)
	if err != nil {
		log.Printf("Failed to create user in DB: %v", err)
		h.sendMessage(chatID, "❌ Ошибка при сохранении данных.")
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

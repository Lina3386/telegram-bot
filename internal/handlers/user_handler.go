package handlers

import (
	"context"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/client"
	"github.com/Lina3386/telegram-bot/internal/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
)

type BotHandler struct {
	bot            *tgbotapi.BotAPI
	authClient     *client.AuthClient
	chatClient     *client.ChatClient
	financeService *services.FinanceService
}

func NewBotHandler(bot *tgbotapi.BotAPI, authClient *client.AuthClient, chatClient *client.ChatClient, financeService *services.FinanceService) *BotHandler {
	return &BotHandler{
		bot:            bot,
		authClient:     authClient,
		chatClient:     chatClient,
		financeService: financeService,
	}
}

func (h *BotHandler) HandleStart(message *tgbotapi.Message) {
	userID := message.From.ID
	username := message.From.UserName
	chatID := message.Chat.ID

	ctx := context.Background()
	log.Printf("User %d (%s) started the bot", userID, username)

	token, err := h.authClient.RegisterTelegramUser(ctx, userID, username)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		h.sendMessage(chatID, "Ошибка регистрации. Попробуйте позже.")
		return
	}

	_, err = h.financeService.CreateUser(ctx, userID, username, token)
	if err != nil {
		log.Printf("Failed to create user in DB: %v", err)
		h.sendMessage(chatID, "Ошибка при сохранении данных.")
		return
	}

	msg := fmt.Sprintf("👋 Добро пожаловать, %s!\n\n"+
		"Я помогу вам управлять финансами.\n\n"+
		"Выберите действие:",
		username,
	)

	h.sendMessageWithKeyboard(chatID, msg, h.mainMenu())
}

// Меню
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

func (h *BotHandler) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	h.bot.Send(msg)
}

func (h *BotHandler) sendMessageWithKeyboard(
	chatID int64,
	text string,
	keyboard tgbotapi.ReplyKeyboardMarkup,
) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	h.bot.Send(msg)
}

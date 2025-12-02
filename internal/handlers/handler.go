package handlers

import (
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

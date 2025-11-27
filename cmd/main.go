package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lina3386/telegram-bot/internal/client"
	"github.com/Lina3386/telegram-bot/internal/config"
	"github.com/Lina3386/telegram-bot/internal/handlers"
	"github.com/Lina3386/telegram-bot/internal/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.LoadConfig()
	log.Println("Config loaded")

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN not set")
	}

	db, err := config.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()
	log.Println("Connected to DB")

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	bot.Debug = cfg.Debug
	log.Printf("Bot authorized: @%s\n", bot.Self.UserName)

	authClient, err := client.NewAuthClient(cfg.AuthServiceURL)
	if err != nil {
		log.Printf("Auth service unavailable: %v", err)
	}
	defer authClient.Close()

	chatClient, err := client.NewChatClient(cfg.ChatServiceURL)
	if err != nil {
		log.Printf("Chat service unavailable: %v", err)
	}
	defer chatClient.Close()

	financeService := services.NewFinanceService(db)
	botHandler := handlers.NewBotHandler(bot, authClient, chatClient, financeService)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	log.Println("Bot is running... (Press Ctrl+C to stop)")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			log.Println("\n Shutting down...")
			return

		case update := <-updates:
			if update.Message != nil {
				if update.Message.IsCommand() {
					switch update.Message.Command() {
					case "start":
						botHandler.HandleStart(update.Message)
					}
				}
			}
			if update.CallbackQuery != nil {

			}
		}
	}
}

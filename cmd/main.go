package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Lina3386/telegram-bot/internal/client"
	"github.com/Lina3386/telegram-bot/internal/config"
	"github.com/Lina3386/telegram-bot/internal/handlers"
	"github.com/Lina3386/telegram-bot/internal/services"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.LoadConfig()
	log.Println("Config loaded")

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN not set")
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

	var authClient *client.AuthClient
	connectedToAuth := false

	for attempt := 1; attempt <= 3; attempt++ {
		var err error
		authClient, err = client.NewAuthClient(cfg.AuthServiceURL)

		if err == nil {
			log.Println("Connected to auth service")
			connectedToAuth = true
			break
		}

		if attempt < 3 {
			log.Printf("Auth service connection attempt %d/3 failed, retrying in 2 seconds...", attempt)
			time.Sleep(2 * time.Second)
		}
	}

	if !connectedToAuth {
		log.Printf("WARNING: Auth service unavailable - running in offline mode")
		log.Printf("User registration will use fallback mode")

		authClient = &client.AuthClient{}
	}

	defer func() {
		if authClient != nil {
			authClient.Close()
		}
	}()

	var chatClient *client.ChatClient
	connectedToChat := false

	for attempt := 1; attempt <= 3; attempt++ {
		var err error
		chatClient, err = client.NewChatClient(cfg.ChatServiceURL)

		if err == nil {
			log.Println("Connected to chat service")
			connectedToChat = true
			break
		}

		if attempt < 3 {
			log.Printf("Chat service connection attempt %d/3 failed, retrying in 2 seconds...", attempt)
			time.Sleep(2 * time.Second)
		}
	}

	if !connectedToChat {
		log.Printf("WARNING: Chat service unavailable")
		chatClient = &client.ChatClient{} // fallback
	}

	defer func() {
		if chatClient != nil {
			chatClient.Close()
		}
	}()

	financeService := services.NewFinanceService(db)
	stateManager := state.NewStateManager()
	botHandler := handlers.NewBotHandler(bot, authClient, chatClient, financeService, stateManager)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	log.Println("Bot is running... (Press Ctrl+C to stop)")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			log.Println("\n Shutting down gracefully...")
			return

		case update := <-updates:
			if update.Message != nil {
				log.Printf(" Message from %d: %s", update.Message.From.ID, update.Message.Text)

				if update.Message.IsCommand() {
					switch update.Message.Command() {
					case "start":
						botHandler.HandleStart(update.Message)
					case "help":
						botHandler.HandleHelp(update.Message)
					case "cancel":
						botHandler.HandleCancel(update.Message)
					default:
						botHandler.HandleUnknownCommand(update.Message)
					}
				} else {
					botHandler.HandleTextMessage(update.Message)
				}
			}

			if update.CallbackQuery != nil {
				log.Printf("Callback from %d: %s", update.CallbackQuery.From.ID, update.CallbackQuery.Data)
				botHandler.HandleCallback(update.CallbackQuery)
			}
		}
	}
}

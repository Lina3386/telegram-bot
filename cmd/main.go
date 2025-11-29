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
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

func main() {
	// ✅ Загружаем конфиг
	cfg := config.LoadConfig()
	log.Println("✅ Config loaded")

	// ✅ Проверяем TOKEN
	if cfg.TelegramToken == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN not set")
	}

	// ✅ Подключаемся к БД
	db, err := config.ConnectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to DB: %v", err)
	}
	defer db.Close()
	log.Println("✅ Connected to DB")

	// ✅ Создаем Telegram бота
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("❌ Failed to create bot: %v", err)
	}
	bot.Debug = cfg.Debug
	log.Printf("✅ Bot authorized: @%s\n", bot.Self.UserName)

	// ✅ Подключаемся к Auth сервису (CRITICAL - проверяем ошибку!)
	authClient, err := client.NewAuthClient(cfg.AuthServiceURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to auth service: %v", err)
	}
	defer authClient.Close()
	log.Println("✅ Connected to auth service")

	// ✅ Подключаемся к Chat сервису (CRITICAL - проверяем ошибку!)
	chatClient, err := client.NewChatClient(cfg.ChatServiceURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to chat service: %v", err)
	}
	defer chatClient.Close()
	log.Println("✅ Connected to chat service")

	// ✅ Инициализируем сервисы
	financeService := services.NewFinanceService(db)
	stateManager := state.NewStateManager()
	botHandler := handlers.NewBotHandler(bot, authClient, chatClient, financeService, stateManager)

	// ✅ Настраиваем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	log.Println("🤖 Bot is running... (Press Ctrl+C to stop)")

	// ✅ Обработчик сигналов для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ✅ ОСНОВНОЙ ЦИКЛ ОБРАБОТКИ
	for {
		select {
		case <-sigChan:
			log.Println("\n⏹️  Shutting down gracefully...")
			return

		case update := <-updates:
			// ✅ Обработка команд
			if update.Message != nil {
				log.Printf("📨 Message from %d: %s", update.Message.From.ID, update.Message.Text)

				// ✅ Обработка команд
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
					// ✅ Обработка текстовых сообщений
					botHandler.HandleTextMessage(update.Message)
				}
			}

			// ✅ Обработка нажатия кнопок (callback queries)
			if update.CallbackQuery != nil {
				log.Printf("🔘 Callback from %d: %s", update.CallbackQuery.From.ID, update.CallbackQuery.Data)
				botHandler.HandleCallback(update.CallbackQuery)
			}
		}
	}
}

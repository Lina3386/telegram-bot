package app

import (
	"context"
	"flag"
	"github.com/Lina3386/telegram-bot/internal/services"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lina3386/telegram-bot/internal/closer"
	"github.com/Lina3386/telegram-bot/internal/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config-path", "../.env", "path to config file")
}

type App struct {
	serviceProvider *ServiceProvider
	bot             *tgbotapi.BotAPI
}

func NewApp(ctx context.Context) (*App, error) {
	a := &App{}

	err := a.initDeps(ctx)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) Run() error {
	defer func() {
		closer.CloseAll()
		closer.Wait()
	}()

	return a.runTelegramBot()
}

func (a *App) initDeps(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initServiceProvider,
		a.initTelegramBot,
		a.initScheduler,
	}

	for i, f := range inits {
		log.Printf("Initializing step %d/%d...", i+1, len(inits))
		err := f(ctx)
		if err != nil {
			log.Printf("Failed at step %d: %v", i+1, err)
			return err
		}
	}
	log.Println("All dependencies initialized")
	return nil
}

func (a *App) initConfig(context.Context) error {
	err := config.Load(configPath)
	if err != nil {
		log.Printf("Config file not found, using environment variables: %v", err)
		// Не возвращаем ошибку, так как переменные могут быть в окружении
	}
	log.Println("Config loaded")
	return nil
}

func (a *App) initServiceProvider(context.Context) error {
	a.serviceProvider = NewServiceProvider()
	log.Println("Service provider created")
	return nil
}

func (a *App) initTelegramBot(ctx context.Context) error {
	bot, err := a.serviceProvider.TelegramBot(ctx)
	if err != nil {
		log.Printf("Failed to initialize bot: %v", err)
		return err
	}
	a.bot = bot
	log.Println("Telegram bot initialized")
	return nil
}

func (a *App) initScheduler(ctx context.Context) error {
	bot := a.bot
	financeService := a.serviceProvider.FinanceService(ctx)
	userRepository := a.serviceProvider.UserRepository(ctx)
	monthlyContribRepository := a.serviceProvider.MonthlyContributionsRepository(ctx)

	scheduler := services.NewScheduler(bot, financeService, userRepository, monthlyContribRepository)
	go func() {
		scheduler.Start(ctx)
	}()
	log.Println("Scheduler started in background")
	return nil
}

func (a *App) runTelegramBot() error {
	log.Println("Telegram bot is starting...")

	_, err := a.bot.GetMe()
	if err != nil {
		log.Printf("Failed to get bot info: %v", err)
		return err
	}
	log.Println("Bot is accessible via Telegram API")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Println("📡 Setting up updates channel...")
	updates := a.bot.GetUpdatesChan(u)

	log.Println("Bot is running and listening for updates... (Press Ctrl+C to stop)")
	log.Println("Try sending /start to the bot to test")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	botHandler := a.serviceProvider.BotHandler(context.Background())

	for {
		select {
		case <-sigChan:
			log.Println("\nShutting down gracefully...")
			return nil

		case update := <-updates:
			if update.Message != nil {
				log.Printf("Message from %d: %s", update.Message.From.ID, update.Message.Text)

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

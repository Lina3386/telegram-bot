package app

import (
	"context"
	"flag"
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
	flag.StringVar(&configPath, "config-path", ".env", "path to config file")
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

	for _, f := range inits {
		err := f(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) initConfig(context.Context) error {
	err := config.Load(configPath)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) initServiceProvider(context.Context) error {
	a.serviceProvider = NewServiceProvider()
	return nil
}

func (a *App) initTelegramBot(ctx context.Context) error {
	bot, err := a.serviceProvider.TelegramBot(ctx)
	if err != nil {
		return err
	}
	a.bot = bot
	return nil
}

func (a *App) initScheduler(ctx context.Context) error {
	return a.serviceProvider.Scheduler(ctx).Start(ctx)
}

func (a *App) runTelegramBot() error {
	log.Println("🤖 Telegram bot is starting...")

	// ✅ Настраиваем обновления
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := a.bot.GetUpdatesChan(u)
	log.Println("🤖 Bot is running... (Press Ctrl+C to stop)")

	// ✅ Обработчик сигналов для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	botHandler := a.serviceProvider.BotHandler(context.Background())

	// ✅ ОСНОВНОЙ ЦИКЛ ОБРАБОТКИ
	for {
		select {
		case <-sigChan:
			log.Println("\n⏹️  Shutting down gracefully...")
			return nil

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


package app

import (
	"context"
	"database/sql"
	"log"

	"github.com/Lina3386/telegram-bot/internal/client"
	"github.com/Lina3386/telegram-bot/internal/client/db"
	"github.com/Lina3386/telegram-bot/internal/client/db/pg"
	"github.com/Lina3386/telegram-bot/internal/closer"
	"github.com/Lina3386/telegram-bot/internal/config"
	"github.com/Lina3386/telegram-bot/internal/config/env"
	"github.com/Lina3386/telegram-bot/internal/handlers"
	"github.com/Lina3386/telegram-bot/internal/repository"
	"github.com/Lina3386/telegram-bot/internal/services"
	"github.com/Lina3386/telegram-bot/internal/state"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

type ServiceProvider struct {
	pgConfig   config.PGConfig
	botConfig  config.BotConfig
	authConfig config.AuthConfig
	chatConfig config.ChatConfig

	dbClient db.Client

	// Repositories
	userRepo    *repository.UserRepository
	incomeRepo  *repository.IncomeRepository
	expenseRepo *repository.ExpenseRepository
	goalRepo    *repository.GoalRepository

	// Services
	financeService *services.FinanceService
	authClient     *client.AuthClient
	chatClient     *client.ChatClient
	scheduler      *services.Scheduler

	// Handlers
	botHandler *handlers.BotHandler

	// State
	stateManager *state.StateManager

	// Bot
	bot *tgbotapi.BotAPI
}

func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{}
}

func (s *ServiceProvider) PGConfig() config.PGConfig {
	if s.pgConfig == nil {
		pgConfig, err := env.NewPGConfig()
		if err != nil {
			log.Fatalf("failed to get pg config: %v", err)
		}
		s.pgConfig = pgConfig
	}
	return s.pgConfig
}

func (s *ServiceProvider) BotConfig() config.BotConfig {
	if s.botConfig == nil {
		botConfig, err := env.NewBotConfig()
		if err != nil {
			log.Fatalf("failed to get bot config: %v", err)
		}
		s.botConfig = botConfig
	}
	return s.botConfig
}

func (s *ServiceProvider) AuthConfig() config.AuthConfig {
	if s.authConfig == nil {
		authConfig, err := env.NewAuthConfig()
		if err != nil {
			log.Fatalf("failed to get auth config: %v", err)
		}
		s.authConfig = authConfig
	}
	return s.authConfig
}

func (s *ServiceProvider) ChatConfig() config.ChatConfig {
	if s.chatConfig == nil {
		chatConfig, err := env.NewChatConfig()
		if err != nil {
			log.Fatalf("failed to get chat config: %v", err)
		}
		s.chatConfig = chatConfig
	}
	return s.chatConfig
}

func (s *ServiceProvider) DBClient(ctx context.Context) db.Client {
	if s.dbClient == nil {
		cl, err := pg.New(ctx, s.PGConfig().DSN())
		if err != nil {
			log.Fatalf("failed to get db client: %v", err)
		}
		err = cl.DB().PingContext(ctx)
		if err != nil {
			log.Fatalf("ping error: %v", err)
		}

		closer.Add(func() error {
			return cl.Close()
		})
		s.dbClient = cl
	}
	return s.dbClient
}

func (s *ServiceProvider) SQLDB(ctx context.Context) *sql.DB {
	return s.DBClient(ctx).DB()
}

func (s *ServiceProvider) UserRepository(ctx context.Context) *repository.UserRepository {
	if s.userRepo == nil {
		s.userRepo = repository.NewUserRepository(s.SQLDB(ctx))
	}
	return s.userRepo
}

func (s *ServiceProvider) IncomeRepository(ctx context.Context) *repository.IncomeRepository {
	if s.incomeRepo == nil {
		s.incomeRepo = repository.NewIncomeRepository(s.SQLDB(ctx))
	}
	return s.incomeRepo
}

func (s *ServiceProvider) ExpenseRepository(ctx context.Context) *repository.ExpenseRepository {
	if s.expenseRepo == nil {
		s.expenseRepo = repository.NewExpenseRepository(s.SQLDB(ctx))
	}
	return s.expenseRepo
}

func (s *ServiceProvider) GoalRepository(ctx context.Context) *repository.GoalRepository {
	if s.goalRepo == nil {
		s.goalRepo = repository.NewGoalRepository(s.SQLDB(ctx))
	}
	return s.goalRepo
}

func (s *ServiceProvider) FinanceService(ctx context.Context) *services.FinanceService {
	if s.financeService == nil {
		s.financeService = services.NewFinanceService(
			s.UserRepository(ctx),
			s.IncomeRepository(ctx),
			s.ExpenseRepository(ctx),
			s.GoalRepository(ctx),
		)
	}
	return s.financeService
}

func (s *ServiceProvider) AuthClient(ctx context.Context) *client.AuthClient {
	if s.authClient == nil {
		authClient, err := client.NewAuthClient(s.AuthConfig().Address())
		if err != nil {
			log.Printf("⚠️  Failed to connect to auth service: %v (will use mock)", err)
			// Не падаем, используем mock
		}
		s.authClient = authClient
		closer.Add(s.authClient.Close)
	}
	return s.authClient
}

func (s *ServiceProvider) ChatClient(ctx context.Context) *client.ChatClient {
	if s.chatClient == nil {
		chatClient, err := client.NewChatClient(s.ChatConfig().Address())
		if err != nil {
			log.Printf("⚠️  Failed to connect to chat service: %v (will use mock)", err)
			// Не падаем, используем mock
		}
		s.chatClient = chatClient
		closer.Add(s.chatClient.Close)
	}
	return s.chatClient
}

func (s *ServiceProvider) StateManager() *state.StateManager {
	if s.stateManager == nil {
		s.stateManager = state.NewStateManager()
	}
	return s.stateManager
}

func (s *ServiceProvider) TelegramBot(ctx context.Context) (*tgbotapi.BotAPI, error) {
	if s.bot == nil {
		token := s.BotConfig().Token()
		if token == "" {
			log.Fatal("❌ TELEGRAM_BOT_TOKEN not set")
		}

		bot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			return nil, err
		}
		bot.Debug = s.BotConfig().Debug()
		log.Printf("✅ Bot authorized: @%s\n", bot.Self.UserName)
		s.bot = bot
	}
	return s.bot, nil
}

func (s *ServiceProvider) Scheduler(ctx context.Context) *services.Scheduler {
	bot, _ := s.TelegramBot(ctx)
	return services.NewScheduler(bot, s.FinanceService(ctx))
}

func (s *ServiceProvider) BotHandler(ctx context.Context) *handlers.BotHandler {
	if s.botHandler == nil {
		s.botHandler = handlers.NewBotHandler(
			s.bot,
			s.FinanceService(ctx),
			s.AuthClient(ctx),
			s.ChatClient(ctx),
			s.StateManager(),
		)
	}
	return s.botHandler
}

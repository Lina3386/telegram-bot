package env

import (
	"errors"
	"github.com/Lina3386/telegram-bot/internal/config"
	"os"
	"strconv"
)

const (
	botTokenEnvName = "TELEGRAM_BOT_TOKEN"
	botDebugEnvName = "LOG_LEVEL"
)

type botConfig struct {
	token string
	debug bool
}

func NewBotConfig() (config.BotConfig, error) {
	token := os.Getenv(botTokenEnvName)
	if token == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN not found")
	}

	debug := os.Getenv(botDebugEnvName) == "debug"

	return &botConfig{
		token: token,
		debug: debug,
	}, nil
}

func (cfg *botConfig) Token() string {
	return cfg.token
}

func (cfg *botConfig) Debug() bool {
	return cfg.debug
}


package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	TelegramToken  string
	DatabaseURL    string
	AuthServiceURL string
	ChatServiceURL string
	Debug          bool
}

func LoadConfig() *Config {
	godotenv.Load()

	return &Config{
		TelegramToken:  getEnv("TELEGRAM_TOKEN", ""),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		AuthServiceURL: getEnv("AUTH_SERVICE_URL", "localhost:50051"),
		ChatServiceURL: getEnv("CHAT_SERVICE_URL", "localhost:50052"),
		Debug:          getEnv("DEBUG", "false") == "true",
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

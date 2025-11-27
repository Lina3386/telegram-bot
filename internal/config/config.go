package config

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
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

	dbURL := getEnv("DATABASE_URL", "")
	log.Printf("DEBUG: DATABASE_URL = %s", dbURL)

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

func ConnectDB(dbURL string) (*sql.DB, error) {
	var db *sql.DB
	var err error
	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 1; i <= maxRetries; i++ {
		log.Printf("Попытка подключения к БД %d/%d...", i, maxRetries)

		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Ошибка открытия подключения: %v", err)
			time.Sleep(retryDelay)
			continue
		}

		// Проверьте подключение
		err = db.Ping()
		if err == nil {
			log.Println("✅ Connected to DB")
			return db, nil
		}

		log.Printf("Ошибка ping БД: %v", err)
		db.Close()

		if i < maxRetries {
			log.Printf("Ждём %v перед следующей попыткой...", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}

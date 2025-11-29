package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken  string
	DatabaseURL    string
	AuthServiceURL string
	ChatServiceURL string
	Debug          bool
}

func LoadConfig() *Config {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found")
	}

	// Явно читаем каждую переменную
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Проверяем, что все переменные установлены
	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)

	return &Config{
		TelegramToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseURL:    dbURL,
		AuthServiceURL: os.Getenv("AUTH_SERVER_URL"),
		ChatServiceURL: os.Getenv("CHAT_SERVER_URL"),
		Debug:          os.Getenv("LOG_LEVEL") == "debug",
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

		// ✅ Проверяем подключение
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

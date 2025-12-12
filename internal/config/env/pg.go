package env

import (
	"errors"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/config"
	"os"
)

const (
	pgUserEnvName     = "DB_USER_TELEGRAM"
	pgPasswordEnvName = "DB_PASSWORD_TELEGRAM"
	pgHostEnvName     = "DB_HOST"
	pgPortEnvName     = "DB_PORT"
	pgNameEnvName     = "DB_NAME_TELEGRAM"
	pgSSLModeEnvName  = "DB_SSLMODE"
)

type pgConfig struct {
	dsn string
}

func NewPGConfig() (config.PGConfig, error) {
	dbUser := os.Getenv(pgUserEnvName)
	dbPassword := os.Getenv(pgPasswordEnvName)
	dbHost := os.Getenv(pgHostEnvName)
	dbPort := os.Getenv(pgPortEnvName)
	dbName := os.Getenv(pgNameEnvName)
	dbSSLMode := os.Getenv(pgSSLModeEnvName)

	if dbHost == "" {
		dbHost = "localhost"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}
	if dbUser == "" || dbPassword == "" || dbName == "" {
		return nil, errors.New("DB_USER, DB_PASSWORD, DB_NAME are required")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode)

	return &pgConfig{
		dsn: dsn,
	}, nil
}

func (cfg *pgConfig) DSN() string {
	return cfg.dsn
}

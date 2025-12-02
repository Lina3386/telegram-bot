package config

import (
	"github.com/joho/godotenv"
)

type PGConfig interface {
	DSN() string
}

type BotConfig interface {
	Token() string
	Debug() bool
}

type AuthConfig interface {
	Address() string
}

type ChatConfig interface {
	Address() string
}

func Load(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}
	return nil
}

package env

import (
	"errors"
	"github.com/Lina3386/telegram-bot/internal/config"
	"net"
	"os"
)

const (
	chatHostEnvName = "CHAT_GRPC_HOST"
	chatPortEnvName = "CHAT_GRPC_PORT"
)

type chatConfig struct {
	host string
	port string
}

func NewChatConfig() (config.ChatConfig, error) {
	host := os.Getenv(chatHostEnvName)
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv(chatPortEnvName)
	if port == "" {
		port = "50052"
	}

	return &chatConfig{
		host: host,
		port: port,
	}, nil
}

func (cfg *chatConfig) Address() string {
	return net.JoinHostPort(cfg.host, cfg.port)
}


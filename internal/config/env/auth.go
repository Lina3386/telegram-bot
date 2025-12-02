package env

import (
	"errors"
	"github.com/Lina3386/telegram-bot/internal/config"
	"net"
	"os"
)

const (
	authHostEnvName = "AUTH_GRPC_HOST"
	authPortEnvName = "AUTH_GRPC_PORT"
)

type authConfig struct {
	host string
	port string
}

func NewAuthConfig() (config.AuthConfig, error) {
	host := os.Getenv(authHostEnvName)
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv(authPortEnvName)
	if port == "" {
		port = "50051"
	}

	return &authConfig{
		host: host,
		port: port,
	}, nil
}

func (cfg *authConfig) Address() string {
	return net.JoinHostPort(cfg.host, cfg.port)
}


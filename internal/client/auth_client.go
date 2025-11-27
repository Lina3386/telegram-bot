package client

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	conn *grpc.ClientConn
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	log.Printf("✅ Connected to auth service at %s", addr)

	return &AuthClient{conn: conn}, nil
}

func (c *AuthClient) RegisterTelegramUser(ctx context.Context, telegramID int64, username string) (string, error) {
	return fmt.Sprintf("token_for_%d", telegramID), nil
}

func (c *AuthClient) VerifyToken(ctx context.Context, token string) (int64, error) {
	return 0, nil
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		log.Println("⏹️ Closing auth service connection")
		return c.conn.Close()
	}
	return nil
}

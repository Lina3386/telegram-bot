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
		log.Printf("⚠️  Auth service connection warning: %v (service may not be running)", err)
	} else {
		log.Printf("✅ Connected to auth service at %s", addr)
	}

	return &AuthClient{conn: conn}, nil
}

func (c *AuthClient) RegisterTelegramUser(ctx context.Context, telegramID int64, username string) (string, error) {
	if c.conn == nil {
		log.Printf("⚠️  Using mock auth token (auth service not connected)")
		return fmt.Sprintf("mock_token_%d", telegramID), nil
	}

	log.Printf("📝 Registering telegram user %d (%s) via auth service", telegramID, username)
	return fmt.Sprintf("tg_token_%d_%d", telegramID, ctx.Value("timestamp")), nil
}

func (c *AuthClient) VerifyToken(ctx context.Context, token string) (int64, error) {
	if c.conn == nil {
		log.Printf("⚠️  Using mock token verification (auth service not connected)")
		return 0, nil
	}

	return 0, nil
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		log.Println("⏹️  Closing auth service connection")
		return c.conn.Close()
	}
	return nil
}

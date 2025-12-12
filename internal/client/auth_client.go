package client

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Lina3386/telegram-bot/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	conn      *grpc.ClientConn
	client    user.UserAPIClient
	available bool
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("Auth service connection warning: %v (service may not be running)", err)
		return &AuthClient{
			conn:      nil,
			client:    nil,
			available: false,
		}, nil
	}

	client := user.NewUserAPIClient(conn)
	log.Printf("Connected to auth service at %s", addr)

	return &AuthClient{
		conn:      conn,
		client:    client,
		available: true,
	}, nil
}

func (c *AuthClient) RegisterTelegramUser(ctx context.Context, telegramID int64, username string) (string, error) {
	if !c.available || c.client == nil {
		log.Printf("Using mock auth token (auth service not connected)")
		return fmt.Sprintf("mock_token_%d", telegramID), nil
	}

	req := &user.RegisterTelegramUserRequest{
		TelegramId: telegramID,
		Username:   username,
	}

	resp, err := c.client.RegisterTelegramUser(ctx, req)
	if err != nil {
		log.Printf("Failed to register telegram user via auth service: %v", err)
		// Fallback to mock
		return fmt.Sprintf("fallback_token_%d", telegramID), nil
	}

	log.Printf("Registered telegram user %d (%s) via auth service", telegramID, username)
	return resp.Token, nil
}

func (c *AuthClient) VerifyToken(ctx context.Context, token string) (int64, error) {
	if !c.available || c.client == nil {
		log.Printf("Using mock token verification (auth service not connected)")
		// Try to extract user ID from token
		if strings.HasPrefix(token, "mock_token_") || strings.HasPrefix(token, "tg_token_") {
			parts := strings.SplitN(token, "_", 2)
			if len(parts) >= 2 {
				if idStr := parts[len(parts)-1]; idStr != "token" {
					if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
						return id, nil
					}
				}
			}
		}
		return 0, fmt.Errorf("mock verification failed")
	}

	req := &user.VerifyTokenRequest{
		Token: token,
	}

	resp, err := c.client.VerifyToken(ctx, req)
	if err != nil {
		log.Printf("Failed to verify token via auth service: %v", err)
		return 0, err
	}

	if !resp.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	return resp.UserId, nil
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		log.Println("🔌 Closing auth service connection")
		return c.conn.Close()
	}
	return nil
}

func (c *AuthClient) IsAvailable() bool {
	return c.available && c.conn != nil
}

package client

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatClient struct {
	conn *grpc.ClientConn
}

func NewChatClient(addr string) (*ChatClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("Chat service connection warning: %v (service may not be running)", err)
	} else {
		log.Printf("Connected to chat service at %s", addr)
	}

	return &ChatClient{conn: conn}, nil
}

func (c *ChatClient) SendMessage(ctx context.Context, chatID int64, from string, message string) error {
	if c.conn == nil {
		log.Printf("Chat service not connected, skipping message log")
		return nil
	}

	log.Printf("[Chat Service] ChatID=%d, From=%s: %s", chatID, from, message)
	return nil
}

func (c *ChatClient) LogFinancialOperation(ctx context.Context, userID int64, operation string, details string) error {
	return c.SendMessage(ctx, userID, "system", fmt.Sprintf("[FINANCE] %s: %s", operation, details))
}

func (c *ChatClient) GetMessage(ctx context.Context, userID int64) ([]string, error) {
	if c.conn == nil {
		log.Printf("Chat service not connected, returning empty messages")
		return []string{}, nil
	}
	return []string{}, nil
}

func (c *ChatClient) Close() error {
	if c.conn != nil {
		log.Println("⏹️  Closing chat service connection")
		return c.conn.Close()
	}
	return nil
}

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
		return nil, fmt.Errorf("failed to connect to chat service: %w", err)
	}

	log.Printf("Connected to chat service at %s", addr)

	return &ChatClient{conn: conn}, nil
}

func (c *ChatClient) SendMessage(ctx context.Context, userID int64, message string) error {
	log.Printf("Message sent to user %d: %s", userID, message)
	return nil
}

func (c *ChatClient) GetMessage(ctx context.Context, userID int64) ([]string, error) {
	return []string{}, nil
}

func (c *ChatClient) Close() error {
	if c.conn != nil {
		log.Println("Closing chat service connection")
		return c.conn.Close()
	}
	return nil
}

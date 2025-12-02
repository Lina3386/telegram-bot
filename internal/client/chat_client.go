package client

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ChatClient struct {
	conn      *grpc.ClientConn
	available bool
}

func NewChatClient(addr string) (*ChatClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	available := err == nil

	if err != nil {
		log.Printf("Chat service connection failed: %v (will use offline mode)", err)
		return &ChatClient{
			conn:      nil,
			available: false,
		}, nil
	}

	log.Printf("Connected to chat service at %s", addr)
	return &ChatClient{
		conn:      conn,
		available: available,
	}, nil
}

func (c *ChatClient) SendMessage(ctx context.Context, userID int64, message string) error {
	if c.conn == nil {
		log.Printf("Chat service not connected, skipping message send")
		return nil
	}

	log.Printf("Message sent to user %d: %s", userID, message)
	return nil
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
		log.Println("Closing chat service connection")
		return c.conn.Close()
	}
	return nil
}

func (c *ChatClient) IsAvailable() bool {
	return c.available && c.conn != nil
}

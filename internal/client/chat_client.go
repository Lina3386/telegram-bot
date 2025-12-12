package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Lina3386/chat-server/pkg/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatClient struct {
	conn      *grpc.ClientConn
	client    chat.ChatApiClient
	available bool
}

func NewChatClient(addr string) (*ChatClient, error) {
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("Chat service connection warning: %v (service may not be running)", err)
		return &ChatClient{
			conn:      nil,
			client:    nil,
			available: false,
		}, nil
	}

	client := chat.NewChatApiClient(conn)
	log.Printf("Connected to chat service at %s", addr)

	return &ChatClient{
		conn:      conn,
		client:    client,
		available: true,
	}, nil
}

func (c *ChatClient) SendMessage(ctx context.Context, chatID int64, from string, message string) error {
	if !c.available || c.client == nil {
		log.Printf("Chat service not connected, skipping message log: %s", message)
		return nil
	}

	req := &chat.SendMessageRequest{
		ChatId:    chatID,
		From:      from,
		Text:      message,
		Timestamp: timestamppb.New(time.Now()),
	}

	_, err := c.client.SendMessage(ctx, req)
	if err != nil {
		log.Printf("Failed to send message via chat service: %v", err)
		return err
	}

	log.Printf("Message sent to chat service: ChatID=%d, From=%s", chatID, from)
	return nil
}

func (c *ChatClient) LogFinancialOperation(ctx context.Context, userID int64, operation string, details string) error {
	return c.SendMessage(ctx, userID, "system", fmt.Sprintf("[FINANCE] %s: %s", operation, details))
}

func (c *ChatClient) GetMessage(ctx context.Context, userID int64) ([]string, error) {
	if !c.available || c.client == nil {
		log.Printf("Chat service not connected, returning empty messages")
		return []string{}, nil
	}

	req := &chat.GetMessagesRequest{
		ChatId: userID,
	}

	resp, err := c.client.GetMessages(ctx, req)
	if err != nil {
		log.Printf("Failed to get messages via chat service: %v", err)
		return []string{}, nil
	}

	var messages []string
	for _, msg := range resp.Messages {
		messages = append(messages, fmt.Sprintf("[%s] %s: %s",
			msg.SentAt.AsTime().Format("15:04"), msg.From, msg.Text))
	}

	return messages, nil
}

func (c *ChatClient) CreateChat(ctx context.Context, userIDs []int64, usernames []string) (int64, error) {
	if !c.available || c.client == nil {
		log.Printf("Chat service not connected, cannot create chat")
		return 0, fmt.Errorf("chat service not available")
	}

	req := &chat.CreateRequest{
		Usernames: usernames,
	}

	resp, err := c.client.Create(ctx, req)
	if err != nil {
		log.Printf("Failed to create chat via chat service: %v", err)
		return 0, err
	}

	log.Printf("Chat created via chat service, ID: %d", resp.Id)
	return resp.Id, nil
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

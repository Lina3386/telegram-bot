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
	// ✅ Для тестирования: просто проверяем подключение
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("⚠️  Chat service connection warning: %v (service may not be running)", err)
		// Не падаем, так как для локального тестирования это нормально
	} else {
		log.Printf("✅ Connected to chat service at %s", addr)
	}

	return &ChatClient{conn: conn}, nil
}

// ✅ SendMessage отправляет сообщение в chat service для логирования
// ПРИМЕЧАНИЕ: Для работы нужен сгенерированный proto код
func (c *ChatClient) SendMessage(ctx context.Context, chatID int64, from string, message string) error {
	if c.conn == nil {
		log.Printf("⚠️  Chat service not connected, skipping message log")
		return nil
	}

	// TODO: После генерации proto кода раскомментировать:
	// client := note.NewChatApiClient(c.conn)
	// _, err := client.SendMessage(ctx, &note.SendMessageRequest{
	//     ChatId: chatID,
	//     From:   from,
	//     Text:   message,
	//     Timestamp: timestamppb.Now(),
	// })
	// if err != nil {
	//     return fmt.Errorf("failed to send message to chat service: %w", err)
	// }

	// Временная реализация - просто логируем
	log.Printf("📤 [Chat Service] ChatID=%d, From=%s: %s", chatID, from, message)
	return nil
}

// ✅ LogFinancialOperation логирует финансовую операцию
func (c *ChatClient) LogFinancialOperation(ctx context.Context, userID int64, operation string, details string) error {
	// Используем userID как chatID для логирования операций пользователя
	return c.SendMessage(ctx, userID, "system", fmt.Sprintf("[FINANCE] %s: %s", operation, details))
}

// ✅ GetMessage получает сообщения для пользователя
func (c *ChatClient) GetMessage(ctx context.Context, userID int64) ([]string, error) {
	if c.conn == nil {
		log.Printf("⚠️  Chat service not connected, returning empty messages")
		return []string{}, nil
	}

	// TODO: Реальный gRPC вызов
	return []string{}, nil
}

// ✅ Close закрывает подключение
func (c *ChatClient) Close() error {
	if c.conn != nil {
		log.Println("⏹️  Closing chat service connection")
		return c.conn.Close()
	}
	return nil
}

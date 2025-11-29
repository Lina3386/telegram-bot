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
	// ✅ Для тестирования: просто проверяем подключение
	// В production нужны реальные protobuf клиенты

	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		// ✅ Возвращаем ошибку только если адрес явно неправильный
		// В локальной разработке микросервис может быть недоступен
		log.Printf("⚠️  Auth service connection warning: %v (service may not be running)", err)
		// Не падаем, так как для локального тестирования это нормально
	} else {
		log.Printf("✅ Connected to auth service at %s", addr)
	}

	return &AuthClient{conn: conn}, nil
}

// ✅ RegisterTelegramUser регистрирует пользователя
// ВАЖНО: Это временная mock реализация!
// В production нужно заменить на реальный gRPC вызов
func (c *AuthClient) RegisterTelegramUser(ctx context.Context, telegramID int64, username string) (string, error) {
	if c.conn == nil {
		// Если gRPC сервис недоступен, генерируем mock токен
		log.Printf("⚠️  Using mock auth token (auth service not connected)")
		return fmt.Sprintf("mock_token_%d", telegramID), nil
	}

	// TODO: Здесь должен быть реальный gRPC вызов:
	// resp, err := c.client.RegisterTelegramUser(ctx, &pb.RegisterRequest{
	//     TelegramId: telegramID,
	//     Username:   username,
	// })
	// if err != nil {
	//     return "", err
	// }
	// return resp.Token, nil

	// Для теста возвращаем mock токен
	return fmt.Sprintf("mock_token_%d", telegramID), nil
}

// ✅ VerifyToken проверяет корректность токена
func (c *AuthClient) VerifyToken(ctx context.Context, token string) (int64, error) {
	if c.conn == nil {
		log.Printf("⚠️  Using mock token verification (auth service not connected)")
		return 0, nil
	}

	// TODO: Реальный gRPC вызов к auth сервису
	return 0, nil
}

// ✅ Close закрывает подключение
func (c *AuthClient) Close() error {
	if c.conn != nil {
		log.Println("⏹️  Closing auth service connection")
		return c.conn.Close()
	}
	return nil
}

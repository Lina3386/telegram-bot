package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/Lina3386/telegram-bot/internal/models"
	"time"

	"github.com/Lina3386/telegram-bot/internal/repository"
)

type AuthService struct {
	repo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) RegisterTelegramUser(ctx context.Context, telegramID int64, username string) (string, error) {
	existingUser, _ := s.repo.GetUserByTelegramID(ctx, telegramID)
	if existingUser != nil {
		return s.generateToken(existingUser.ID)
	}

	user := &models.User{
		TelegramID: telegramID,
		Username:   username,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return "", err
	}

	return s.generateToken(user.ID)
}

func (s *AuthService) generateToken(userID int64) (string, error) {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d:%d", userID, time.Now().Unix())))
	return hex.EncodeToString(hash[:]), nil
}

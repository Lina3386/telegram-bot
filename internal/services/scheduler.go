package services

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Scheduler struct {
	bot            *tgbotapi.BotAPI
	financeService *FinanceService
}

func NewScheduler(bot *tgbotapi.BotAPI, financeService *FinanceService) *Scheduler {
	return &Scheduler{
		bot:            bot,
		financeService: financeService,
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Hour) // Проверяем каждый час
	defer ticker.Stop()
	s.checkPayDates(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  Scheduler stopped")
			return nil
		case <-ticker.C:
			s.checkPayDates(ctx)
		}
	}
	return nil
}

func (s *Scheduler) checkPayDates(ctx context.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	incomes, err := s.financeService.GetIncomesByPayDate(ctx, today)
	if err != nil {
		log.Printf("Failed to get incomes by pay date: %v", err)
		return
	}

	for _, income := range incomes {
		telegramID := income.UserID
		goals, err := s.financeService.GetUserActiveGoalsByTelegramID(ctx, telegramID)
		if err != nil {
			log.Printf("Failed to get goals for user %d: %v", telegramID, err)
			continue
		}

		if len(goals) == 0 {
			msg := fmt.Sprintf("💰 Сегодня день получки!\n\n%s: %d₽", income.Name, income.Amount)
			s.sendNotification(telegramID, msg)
		} else {
			for _, goal := range goals {
				msg := fmt.Sprintf(
					"💰 Сегодня день получки!\n\n%s: %d₽\n\n"+
						"🎯 Цель: %s\n"+
						"Отложите: %d₽",
					income.Name, income.Amount, goal.GoalName, goal.MonthlyContrib,
				)
				s.sendNotificationWithButtons(telegramID, msg, goal.ID, goal.MonthlyContrib)
			}
		}

		nextPayDate := today.AddDate(0, 1, 0)
		err = s.financeService.UpdateIncomeNextPayDate(ctx, income.ID, nextPayDate)
		if err != nil {
			log.Printf("Failed to update next pay date for income %d: %v", income.ID, err)
		}
	}
}

func (s *Scheduler) sendNotification(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send notification to %d: %v", chatID, err)
	}
}

func (s *Scheduler) sendNotificationWithButtons(chatID int64, text string, goalID int64, amount int64) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("✅ Добавить %d₽", amount),
				fmt.Sprintf("add_contribution_%d_%d", goalID, amount),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"➖ Вычесть",
				fmt.Sprintf("withdraw_%d", goalID),
			),
		),
	)
	_, err := s.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send notification with buttons to %d: %v", chatID, err)
	}
}

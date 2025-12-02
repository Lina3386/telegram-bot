package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Lina3386/telegram-bot/internal/models"
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

// ✅ Start запускает фоновую задачу для проверки дат получки
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Проверяем каждый час
	defer ticker.Stop()

	// ✅ Проверяем сразу при запуске
	s.checkPayDates(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  Scheduler stopped")
			return
		case <-ticker.C:
			s.checkPayDates(ctx)
		}
	}
}

// ✅ checkPayDates проверяет доходы с сегодняшней датой получки
func (s *Scheduler) checkPayDates(ctx context.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// ✅ Получаем всех пользователей с доходами на сегодня
	incomes, err := s.financeService.GetIncomesByPayDate(ctx, today)
	if err != nil {
		log.Printf("Failed to get incomes by pay date: %v", err)
		return
	}

	for _, income := range incomes {
		// ✅ income.UserID теперь содержит telegram_id (изменено в репозитории)
		telegramID := income.UserID

		// ✅ Получаем активные цели пользователя (по telegram_id)
		goals, err := s.financeService.GetUserActiveGoalsByTelegramID(ctx, telegramID)
		if err != nil {
			log.Printf("Failed to get goals for user %d: %v", telegramID, err)
			continue
		}

		if len(goals) == 0 {
			// ✅ Нет целей - просто уведомляем о получке
			msg := fmt.Sprintf("💰 Сегодня день получки!\n\n%s: %d₽", income.Name, income.Amount)
			s.sendNotification(telegramID, msg)
		} else {
			// ✅ Отправляем уведомление с кнопками для каждой цели
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

		// ✅ Обновляем next_pay_date на следующий месяц
		nextPayDate := today.AddDate(0, 1, 0)
		err = s.financeService.UpdateIncomeNextPayDate(ctx, income.ID, nextPayDate)
		if err != nil {
			log.Printf("Failed to update next pay date for income %d: %v", income.ID, err)
		}
	}
}

// ✅ sendNotification отправляет простое уведомление
func (s *Scheduler) sendNotification(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := s.bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send notification to %d: %v", chatID, err)
	}
}

// ✅ sendNotificationWithButtons отправляет уведомление с кнопками
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


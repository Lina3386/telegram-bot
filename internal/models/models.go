package models

import "time"

// User - пользователь
type User struct {
	ID             int64
	TelegramID     int64
	Username       string
	AuthToken      string
	MonthlyExpense int64 // Расходы на жизнь
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Income - источник дохода
type Income struct {
	ID           int64
	UserID       int64
	Name         string // "Зарплата", "Пособие"
	Amount       int64
	RecurringDay int // День месяца
	NextPayDate  time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Expense - расходы
type Expense struct {
	ID        int64
	UserID    int64
	Name      string // Аренда, еда
	Amount    int64  // Месячный бюджет
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SavingsGoal - цель накопления
type SavingsGoal struct {
	ID             int64
	UserID         int64
	GoalName       string
	TargetAmount   int64
	CurrentAmount  int64
	MonthlyContrib int64
	TargetDate     time.Time
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Состояние диалога
type UserState struct {
	UserID   int64
	State    string
	TempDate map[string]string
}

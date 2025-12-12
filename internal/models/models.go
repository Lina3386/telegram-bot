package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID             int64  `db:"id"`
	TelegramID     int64  `db:"telegram_id"`
	Username       string `db:"username"`
	AuthToken      string `db:"auth_token"`
	MonthlyExpense int64
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type Income struct {
	ID               int64     `db:"id"`
	UserID           int64     `db:"user_id"`
	Name             string    `db:"name"`
	Amount           int64     `db:"amount"`
	Frequency        string    `db:"frequency"`
	RecurringDay     int       `db:"recurring_day"`
	NotificationHour int       `db:"notification_hour"`
	NextPayDate      time.Time `db:"next_pay_date"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

type Expense struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Name      string    `db:"name"`
	Amount    int64     `db:"amount"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// цель накопления
type SavingsGoal struct {
	ID                 int64        `db:"id"`
	UserID             int64        `db:"user_id"`
	GoalName           string       `db:"goal_name"`
	TargetAmount       int64        `db:"target_amount"`
	CurrentAmount      int64        `db:"current_amount"`
	MonthlyContrib     int64        `db:"monthly_contrib"`
	MonthlyBudgetLimit int64        `db:"monthly_budget_limit"`
	MonthlyAccumulated int64        `db:"monthly_accumulated"`
	MonthStarted       sql.NullTime `db:"month_started"`
	TargetDate         time.Time    `db:"target_date"`
	Priority           int          `db:"priority"`
	Status             string       `db:"status"`
	CreatedAt          time.Time    `db:"created_at"`
	UpdatedAt          time.Time    `db:"updated_at"`
}

type MonthlyContribution struct {
	ID                int64     `db:"id"`
	UserID            int64     `db:"user_id"`
	GoalID            int64     `db:"goal_id"`
	Month             time.Time `db:"month"`
	AmountContributed int64     `db:"amount"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type IncomeProcessingLog struct {
	ID            int64
	IncomeID      int64
	UserID        int64
	ProcessedDate time.Time
	IncomeAmount  int64
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UserState struct {
	UserID   int64
	State    string
	TempDate map[string]string
}

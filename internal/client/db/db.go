package db

import (
	"database/sql"
)

// Client - интерфейс для работы с БД
type Client interface {
	DB() *sql.DB
	Close() error
}



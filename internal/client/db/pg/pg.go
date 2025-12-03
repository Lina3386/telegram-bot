package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Lina3386/telegram-bot/internal/client/db"
	_ "github.com/lib/pq"
)

type pgClient struct {
	db *sql.DB
}

func New(ctx context.Context, dsn string) (db.Client, error) {
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &pgClient{
		db: sqlDB,
	}, nil
}

func (c *pgClient) DB() *sql.DB {
	return c.db
}

func (c *pgClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}



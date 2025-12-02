package main

import (
	"context"
	"github.com/Lina3386/telegram-bot/internal/app"
	"log"
)

func main() {

	ctx := context.Background()

	a, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}

	err = a.Run()
	if err != nil {
		log.Fatalf("failed to run app: %v", err)

	}
}

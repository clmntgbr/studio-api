package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/fetch"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := fetch.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("worker-fetch config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := fetch.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-fetch stopped: %v", err)
	}
}

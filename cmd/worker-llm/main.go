package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/llm"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg, err := llm.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := llm.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-llm stopped: %v", err)
	}
}

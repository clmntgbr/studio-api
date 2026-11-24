package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/silence"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg, err := silence.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := silence.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-silence stopped: %v", err)
	}
}

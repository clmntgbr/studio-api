package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/transcribe"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg, err := transcribe.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := transcribe.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-transcribe stopped: %v", err)
	}
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"go-api/internal/renderworker"
)

func main() {
	_ = godotenv.Load()
	cfg, err := renderworker.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := renderworker.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-render stopped: %v", err)
	}
}

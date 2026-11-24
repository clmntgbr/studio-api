package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/media"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := media.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("worker-media config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := media.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-media stopped: %v", err)
	}
}

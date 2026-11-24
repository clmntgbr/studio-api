package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/textdetect"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg, err := textdetect.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := textdetect.Run(ctx, cfg); err != nil {
		log.Fatalf("worker-text stopped: %v", err)
	}
}

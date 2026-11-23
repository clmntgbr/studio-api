package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/ingest"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := ingest.LoadWorkerConfigFromEnv()
	if err != nil {
		log.Fatalf("worker-ingest config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := ingest.RunWorker(ctx, cfg); err != nil {
		log.Fatalf("worker-ingest stopped: %v", err)
	}
}

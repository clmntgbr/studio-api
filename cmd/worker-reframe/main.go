package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-api/internal/workerstub"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := workerstub.Run(ctx, "reframe"); err != nil {
		log.Fatalf("worker-reframe stopped: %v", err)
	}
}

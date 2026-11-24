package workerstub

import (
	"context"
	"fmt"
	"log"
	"os"

	"go-api/internal/studio/topology"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Run connects to RabbitMQ, consumes durable queue q.<name>, logs each body, and acks.
// No database or object-storage access — scaffolding only.
func Run(ctx context.Context, name string) error {
	_ = godotenv.Load()

	if name == "" {
		return fmt.Errorf("worker name is required")
	}

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		return fmt.Errorf("RABBITMQ_URL is required")
	}

	queue := "q." + name

	conn, err := amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	defer ch.Close()

	if err := topology.DeclareExchanges(ch); err != nil {
		return err
	}
	if err := topology.DeclareJobQueue(ch, name, 30000); err != nil {
		return err
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", queue, err)
	}

	log.Printf("worker-%s started, consuming %s", name, queue)

	for {
		select {
		case <-ctx.Done():
			log.Printf("worker-%s shutting down", name)
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("worker-%s: rabbitmq deliveries channel closed", name)
			}
			log.Printf("worker-%s received message (%d bytes): %s", name, len(delivery.Body), string(delivery.Body))
			if err := delivery.Ack(false); err != nil {
				return fmt.Errorf("worker-%s ack: %w", name, err)
			}
		}
	}
}

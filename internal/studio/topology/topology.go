package topology

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeJobs     = "studio.jobs"
	ExchangeResults  = "studio.results"
	ExchangeProgress = "studio.progress"
	ExchangeRetry    = "studio.retry"
	ExchangeEvents   = "studio.events"
)

// DeclareExchanges creates the studio.* exchanges.
func DeclareExchanges(ch *amqp.Channel) error {
	decls := []struct {
		name string
		kind string
	}{
		{ExchangeJobs, "direct"},
		{ExchangeResults, "topic"},
		{ExchangeProgress, "fanout"},
		{ExchangeRetry, "direct"},
		{ExchangeEvents, "topic"},
	}
	for _, d := range decls {
		if err := ch.ExchangeDeclare(d.name, d.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare exchange %s: %w", d.name, err)
		}
	}
	return nil
}

// DeclareJobQueue declares q.<job>, q.<job>.retry, q.<job>.dlq and bindings.
// Retry uses a fixed TTL (default 30s) DLX back to the main queue.
func DeclareJobQueue(ch *amqp.Channel, jobType string, retryTTLMs int) error {
	if jobType == "" {
		return fmt.Errorf("job type is required")
	}
	if retryTTLMs <= 0 {
		retryTTLMs = 30000
	}
	main := "q." + jobType
	retry := main + ".retry"
	dlq := main + ".dlq"

	if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare %s: %w", dlq, err)
	}

	if _, err := ch.QueueDeclare(main, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": dlq,
	}); err != nil {
		return fmt.Errorf("declare %s: %w", main, err)
	}

	if _, err := ch.QueueDeclare(retry, true, false, false, false, amqp.Table{
		"x-message-ttl":             int32(retryTTLMs),
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": main,
	}); err != nil {
		return fmt.Errorf("declare %s: %w", retry, err)
	}

	if err := ch.QueueBind(main, "job."+jobType, ExchangeJobs, false, nil); err != nil {
		return fmt.Errorf("bind %s: %w", main, err)
	}
	if err := ch.QueueBind(retry, "retry."+jobType, ExchangeRetry, false, nil); err != nil {
		return fmt.Errorf("bind %s: %w", retry, err)
	}
	return nil
}

// Bus serializes channel publish/ack (amqp091.Channel is not concurrency-safe).
type Bus struct {
	ch       *amqp.Channel
	confirms <-chan amqp.Confirmation
	mu       sync.Mutex
}

func NewBus(ch *amqp.Channel, confirmBuffer int) (*Bus, error) {
	if confirmBuffer <= 0 {
		confirmBuffer = 8
	}
	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("confirm mode: %w", err)
	}
	return &Bus{
		ch:       ch,
		confirms: ch.NotifyPublish(make(chan amqp.Confirmation, confirmBuffer)),
	}, nil
}

func (b *Bus) Ack(d amqp.Delivery) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return d.Ack(false)
}

func (b *Bus) Nack(d amqp.Delivery) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return d.Nack(false, false)
}

func (b *Bus) Publish(ctx context.Context, exchange, routingKey string, msg amqp.Publishing) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ch.PublishWithContext(ctx, exchange, routingKey, false, false, msg); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c, ok := <-b.confirms:
		if !ok {
			return fmt.Errorf("confirm channel closed")
		}
		if !c.Ack {
			return fmt.Errorf("publish not confirmed exchange=%s key=%s", exchange, routingKey)
		}
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("publish confirm timeout exchange=%s key=%s", exchange, routingKey)
	}
}

func (b *Bus) PublishJob(ctx context.Context, routingKey string, body []byte, messageID, correlationID string) error {
	return b.Publish(ctx, ExchangeJobs, routingKey, amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		MessageId:     messageID,
		CorrelationId: correlationID,
		Body:          body,
	})
}

func (b *Bus) PublishResult(ctx context.Context, routingKey string, body []byte, messageID, correlationID string) error {
	return b.Publish(ctx, ExchangeResults, routingKey, amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		MessageId:     messageID,
		CorrelationId: correlationID,
		Body:          body,
	})
}

func (b *Bus) PublishProgress(ctx context.Context, body []byte, messageID, correlationID string) error {
	return b.Publish(ctx, ExchangeProgress, "", amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Transient,
		MessageId:     messageID,
		CorrelationId: correlationID,
		Body:          body,
	})
}

package studiojobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"go-api/internal/application/realtime"
	"go-api/internal/domain/port"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ProgressConsumer fans studio.progress to Centrifugo projects:{id}.
// Progress messages are ephemeral: failures are logged and acked (never block the pipeline).
type ProgressConsumer struct {
	url       string
	media     mediaProjectResolver
	publisher *realtime.Publisher
}

type mediaProjectResolver interface {
	ProjectIDForMedia(ctx context.Context, mediaID uuid.UUID) (uuid.UUID, error)
}

func NewProgressConsumer(rabbitURL string, media mediaProjectResolver, transport port.RealtimePublisher) *ProgressConsumer {
	return &ProgressConsumer{
		url:       rabbitURL,
		media:     media,
		publisher: realtime.NewPublisher(transport),
	}
}

func (c *ProgressConsumer) Start(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := topology.DeclareExchanges(ch); err != nil {
		return err
	}
	q, err := ch.QueueDeclare("q.progress", true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := ch.QueueBind(q.Name, "", topology.ExchangeProgress, false, nil); err != nil {
		return err
	}
	if err := ch.Qos(16, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume(q.Name, "core-progress", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("studio progress consumer started (q.progress)")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("progress channel closed")
			}
			_ = c.handle(ctx, d)
			_ = d.Ack(false)
		}
	}
}

func (c *ProgressConsumer) handle(ctx context.Context, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		return err
	}
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return err
	}
	projectID, err := c.media.ProjectIDForMedia(ctx, mediaID)
	if err != nil || projectID == uuid.Nil {
		return err
	}
	var pp envelope.ProgressPayload
	_ = json.Unmarshal(msg.Payload, &pp)
	step := strings.TrimPrefix(msg.Type, "progress.")
	return c.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionProgress, projectID.String(), map[string]any{
		"mediaId": mediaID.String(),
		"step":    step,
		"percent": pp.Pct,
		"stage":   pp.Stage,
		"message": pp.Message,
	})
}

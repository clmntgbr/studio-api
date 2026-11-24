package studiojobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/realtime"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"
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
	renders   renderProjectResolver
	renderRepo domainrender.WriteRepository
	steps     domainmedia.WriteRepository
	publisher *realtime.Publisher
}

type mediaProjectResolver interface {
	ProjectIDForMedia(ctx context.Context, mediaID uuid.UUID) (uuid.UUID, error)
}

type renderProjectResolver interface {
	ProjectIDForRender(ctx context.Context, renderID uuid.UUID) (uuid.UUID, error)
}

func NewProgressConsumer(
	rabbitURL string,
	media mediaProjectResolver,
	renders renderProjectResolver,
	renderRepo domainrender.WriteRepository,
	steps domainmedia.WriteRepository,
	transport port.RealtimePublisher,
) *ProgressConsumer {
	return &ProgressConsumer{
		url:        rabbitURL,
		media:      media,
		renders:    renders,
		renderRepo: renderRepo,
		steps:      steps,
		publisher:  realtime.NewPublisher(transport),
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
	corrID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return err
	}
	var pp envelope.ProgressPayload
	_ = json.Unmarshal(msg.Payload, &pp)
	stepName := strings.TrimPrefix(msg.Type, "progress.")

	if stepName == "render" {
		return c.handleRenderProgress(ctx, corrID, pp)
	}

	if c.steps != nil {
		if kind, ok := progressStepKind(stepName); ok {
			_ = mediacmd.MarkStepRunning(ctx, c.steps, corrID, kind, time.Now().UTC())
		}
	}
	projectID, err := c.media.ProjectIDForMedia(ctx, corrID)
	if err != nil || projectID == uuid.Nil {
		return err
	}
	return c.publisher.ToProject(ctx, realtime.EntityMedia, realtime.ActionProgress, projectID.String(), map[string]any{
		"mediaId": corrID.String(),
		"step":    stepName,
		"percent": pp.Pct,
		"stage":   pp.Stage,
		"message": pp.Message,
	})
}

func (c *ProgressConsumer) handleRenderProgress(ctx context.Context, renderID uuid.UUID, pp envelope.ProgressPayload) error {
	if c.renders == nil {
		return nil
	}
	projectID, err := c.renders.ProjectIDForRender(ctx, renderID)
	if err != nil || projectID == uuid.Nil {
		return err
	}
	if c.renderRepo != nil {
		if r, err := c.renderRepo.GetByID(ctx, renderID); err == nil && r != nil {
			if r.Status != domainrender.StatusCompleted &&
				r.Status != domainrender.StatusFailed &&
				r.Status != domainrender.StatusCancelled {
				now := time.Now().UTC()
				if r.Status == domainrender.StatusQueued || r.Status == domainrender.StatusPending {
					r.Status = domainrender.StatusRunning
					if r.StartedAt == nil {
						r.StartedAt = &now
					}
				}
				if pp.Pct > r.Progress {
					r.Progress = pp.Pct
				}
				_ = c.renderRepo.Update(ctx, r)
			}
		}
	}
	return c.publisher.ToProject(ctx, realtime.EntityRender, realtime.ActionRenderProgress, projectID.String(), map[string]any{
		"renderId": renderID.String(),
		"percent":  pp.Pct,
		"stage":    pp.Stage,
		"message":  pp.Message,
	})
}

func progressStepKind(name string) (domainmedia.StepKind, bool) {
	switch domainmedia.StepKind(name) {
	case domainmedia.StepFetch,
		domainmedia.StepProbe,
		domainmedia.StepAudio,
		domainmedia.StepProxy,
		domainmedia.StepWaveform,
		domainmedia.StepSilence,
		domainmedia.StepTranscribe,
		domainmedia.StepText,
		domainmedia.StepLLM,
		domainmedia.StepScenes,
		domainmedia.StepReframe:
		return domainmedia.StepKind(name), true
	default:
		return "", false
	}
}

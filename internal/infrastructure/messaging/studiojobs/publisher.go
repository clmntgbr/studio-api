package studiojobs

import (
	"context"
	"fmt"

	"go-api/internal/domain/port"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	bus *topology.Bus
}

func NewPublisher(rabbitURL string) (*Publisher, error) {
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := topology.DeclareExchanges(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	for _, job := range []string{"fetch", "probe", "audio", "proxy", "waveform"} {
		if err := topology.DeclareJobQueue(ch, job, 30000); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, err
		}
	}
	bus, err := topology.NewBus(ch, 8)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	// Keep conn alive via channel; leak conn intentionally for process lifetime.
	_ = conn
	return &Publisher{bus: bus}, nil
}

func (p *Publisher) PublishFetchJob(ctx context.Context, mediaID, rawURL string) error {
	msg, body, err := envelope.NewJob("fetch", mediaID, "", 1, map[string]any{
		"mediaId": mediaID,
		"url":     rawURL,
	})
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.fetch", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishProbeJob(ctx context.Context, mediaID, sourceKey string) error {
	msg, body, err := envelope.NewJob("probe", mediaID, "", 1, map[string]any{
		"mediaId":   mediaID,
		"sourceKey": sourceKey,
	})
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.probe", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishAudioJob(ctx context.Context, mediaID, sourceKey string) error {
	msg, body, err := envelope.NewJob("audio", mediaID, "", 1, map[string]any{
		"mediaId":   mediaID,
		"sourceKey": sourceKey,
	})
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.audio", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishProxyJob(ctx context.Context, mediaID, sourceKey string) error {
	msg, body, err := envelope.NewJob("proxy", mediaID, "", 1, map[string]any{
		"mediaId":   mediaID,
		"sourceKey": sourceKey,
		"proxy": map[string]any{
			"height":    720,
			"gopFrames": 15,
			"crf":       23,
		},
	})
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.proxy", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishWaveformJob(ctx context.Context, mediaID, audioKey string) error {
	msg, body, err := envelope.NewJob("waveform", mediaID, "", 1, map[string]any{
		"mediaId":  mediaID,
		"audioKey": audioKey,
	})
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.waveform", body, msg.MessageID, mediaID)
}

var _ port.JobPublisher = (*Publisher)(nil)

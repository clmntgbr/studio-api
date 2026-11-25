package studiojobs

import (
	"context"
	"encoding/json"
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
	// Retry TTLs must match each worker's DeclareJobQueue (RabbitMQ rejects inequivalent args).
	jobRetryTTLMs := map[string]int{
		"fetch": 30000, "probe": 30000, "audio": 30000, "proxy": 30000, "waveform": 30000,
		"silence": 30000, "transcribe": 30000, "text": 30000,
		"llm": 300000, "scenes": 30000, "reframe": 30000,
		"render": 7200000,
	}
	for job, ttl := range jobRetryTTLMs {
		if err := topology.DeclareJobQueue(ch, job, ttl); err != nil {
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

func (p *Publisher) PublishSilenceJob(ctx context.Context, mediaID, audioKey string, opts port.SilenceJobOptions) error {
	if opts.ThresholdMode == "" {
		opts.ThresholdMode = "auto"
	}
	if opts.NoiseFloorOffsetDb == 0 {
		opts.NoiseFloorOffsetDb = 12
	}
	attempt := opts.Attempt
	if attempt < 1 {
		attempt = 1
	}
	payload := map[string]any{
		"mediaId":            mediaID,
		"audioKey":           audioKey,
		"thresholdMode":      opts.ThresholdMode,
		"noiseFloorOffsetDb": opts.NoiseFloorOffsetDb,
	}
	if opts.SilenceThresholdDb != nil {
		payload["silenceThresholdDb"] = *opts.SilenceThresholdDb
	}
	msg, body, err := envelope.NewJob("silence", mediaID, "", attempt, payload)
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.silence", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishTranscribeJob(ctx context.Context, mediaID, audioKey string, opts port.TranscribeJobOptions) error {
	payload := map[string]any{
		"mediaId":  mediaID,
		"audioKey": audioKey,
	}
	if opts.Language != nil && *opts.Language != "" {
		payload["language"] = *opts.Language
	}
	msg, body, err := envelope.NewJob("transcribe", mediaID, "", 1, payload)
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.transcribe", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishRenderJob(ctx context.Context, renderID string, plan json.RawMessage) error {
	payload := map[string]any{}
	if len(plan) > 0 {
		_ = json.Unmarshal(plan, &payload)
	}
	payload["renderId"] = renderID
	msg, body, err := envelope.NewJob("render", renderID, "", 1, payload)
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.render", body, msg.MessageID, renderID)
}

func (p *Publisher) PublishTextJob(ctx context.Context, mediaID, wordsKey string, opts port.TextJobOptions) error {
	if opts.Language == "" {
		opts.Language = "fr"
	}
	if opts.DetectionLevel == "" {
		opts.DetectionLevel = "aggressive"
	}
	attempt := opts.Attempt
	if attempt < 1 {
		attempt = 1
	}
	payload := map[string]any{
		"mediaId":              mediaID,
		"wordsKey":             wordsKey,
		"language":             opts.Language,
		"detectionLevel":       opts.DetectionLevel,
		"fillersEnabled":       opts.FillersEnabled,
		"repetitionsEnabled":   opts.RepetitionsEnabled,
		"falseStartsEnabled":   opts.FalseStartsEnabled,
	}
	msg, body, err := envelope.NewJob("text", mediaID, "", attempt, payload)
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.text", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishLLMJob(ctx context.Context, mediaID, wordsKey string, opts port.LLMJobOptions) error {
	if opts.Language == "" {
		opts.Language = "fr"
	}
	tasks := opts.Tasks
	if len(tasks) == 0 {
		tasks = []string{"sponsors", "chapters", "viral"}
	}
	payload := map[string]any{
		"mediaId":        mediaID,
		"wordsKey":       wordsKey,
		"language":       opts.Language,
		"tasks":          tasks,
		"sponsorsAction": opts.SponsorsAction,
	}
	msg, body, err := envelope.NewJob("llm", mediaID, "", 1, payload)
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.llm", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishReframeJob(ctx context.Context, mediaID, proxyKey string, targetWidth, targetHeight int) error {
	payload := map[string]any{
		"mediaId":  mediaID,
		"proxyKey": proxyKey,
	}
	if targetWidth > 0 && targetHeight > 0 {
		payload["targetWidth"] = targetWidth
		payload["targetHeight"] = targetHeight
	}
	msg, body, err := envelope.NewJob("reframe", mediaID, "", 1, payload)
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.reframe", body, msg.MessageID, mediaID)
}

func (p *Publisher) PublishScenesJob(ctx context.Context, mediaID, proxyKey string, attempt int) error {
	if attempt < 1 {
		attempt = 1
	}
	msg, body, err := envelope.NewJob("scenes", mediaID, "", attempt, map[string]any{
		"mediaId":  mediaID,
		"proxyKey": proxyKey,
	})
	if err != nil {
		return err
	}
	return p.bus.PublishJob(ctx, "job.scenes", body, msg.MessageID, mediaID)
}

var _ port.JobPublisher = (*Publisher)(nil)

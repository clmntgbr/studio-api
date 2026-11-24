package studiojobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ResultsConsumer consumes studio.results (q.results) and runs shared post-source orchestration.
type ResultsConsumer struct {
	url         string
	markReady   *mediacmd.MarkSourceReadyHandler
	markFail    *mediacmd.MarkFetchFailedHandler
	applyProbe  *mediacmd.ApplyProbeResultHandler
	applyAudio  *mediacmd.ApplyAudioResultHandler
	applyProxy  *mediacmd.ApplyProxyResultHandler
	applyWave   *mediacmd.ApplyWaveformResultHandler
}

func NewResultsConsumer(
	rabbitURL string,
	markReady *mediacmd.MarkSourceReadyHandler,
	markFail *mediacmd.MarkFetchFailedHandler,
	applyProbe *mediacmd.ApplyProbeResultHandler,
	applyAudio *mediacmd.ApplyAudioResultHandler,
	applyProxy *mediacmd.ApplyProxyResultHandler,
	applyWave *mediacmd.ApplyWaveformResultHandler,
) *ResultsConsumer {
	return &ResultsConsumer{
		url:        rabbitURL,
		markReady:  markReady,
		markFail:   markFail,
		applyProbe: applyProbe,
		applyAudio: applyAudio,
		applyProxy: applyProxy,
		applyWave:  applyWave,
	}
}

func (c *ResultsConsumer) Start(ctx context.Context) error {
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
	if err := topology.DeclareResultsQueue(ch, 30000); err != nil {
		return err
	}
	if err := ch.Qos(8, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume("q.results", "core-results", false, false, false, false, nil)
	if err != nil {
		return err
	}

	bus, err := topology.NewBus(ch, 8)
	if err != nil {
		return err
	}

	log.Println("studio results consumer started (q.results)")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("results channel closed")
			}
			if err := c.handle(ctx, bus, d); err != nil {
				log.Printf("results handle error: %v", err)
			}
		}
	}
}

func (c *ResultsConsumer) handle(ctx context.Context, bus *topology.Bus, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		log.Printf("poison results envelope: %v", err)
		return bus.Nack(d)
	}

	var handleErr error
	switch msg.Type {
	case "result.fetch.succeeded":
		handleErr = c.onFetchSucceeded(ctx, msg)
	case "result.fetch.failed":
		handleErr = c.onFetchFailed(ctx, msg)
	case "result.probe.succeeded":
		handleErr = c.onProbeSucceeded(ctx, msg)
	case "result.audio.succeeded":
		handleErr = c.onAudioSucceeded(ctx, msg)
	case "result.proxy.succeeded":
		handleErr = c.onProxySucceeded(ctx, msg)
	case "result.waveform.succeeded":
		handleErr = c.onWaveformSucceeded(ctx, msg)
	case "result.probe.failed", "result.audio.failed", "result.proxy.failed", "result.waveform.failed":
		log.Printf("pipeline step failed type=%s correlationId=%s", msg.Type, msg.CorrelationID)
		handleErr = nil
	default:
		return bus.Ack(d)
	}
	if handleErr != nil {
		log.Printf("results handling failed type=%s: %v", msg.Type, handleErr)
		return bus.Nack(d)
	}
	return bus.Ack(d)
}

type fetchOutput struct {
	SourceKey string `json:"sourceKey"`
	SizeBytes int64  `json:"sizeBytes"`
	Title     string `json:"title"`
}

type probeOutput struct {
	DurationMs    int64   `json:"durationMs"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	FPSNum        int     `json:"fpsNum"`
	FPSDen        int     `json:"fpsDen"`
	VideoCodec    string  `json:"videoCodec"`
	AudioCodec    string  `json:"audioCodec"`
	AudioChannels int     `json:"audioChannels"`
}

type audioOutput struct {
	AnalysisKey      string `json:"analysisKey"`
	TranscriptionKey string `json:"transcriptionKey"`
	MasterKey        string `json:"masterKey"`
}

type proxyOutput struct {
	ProxyKey string `json:"proxyKey"`
}

type waveformOutput struct {
	PeaksKey string `json:"peaksKey"`
}

func (c *ResultsConsumer) onFetchSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[fetchOutput](msg)
	if err != nil {
		return err
	}
	var sizePtr *int64
	if out.SizeBytes > 0 {
		sz := out.SizeBytes
		sizePtr = &sz
	}
	var titlePtr *string
	if out.Title != "" {
		t := out.Title
		titlePtr = &t
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.markReady.Handle(hCtx, mediacmd.MarkSourceReadyCommand{
		MediaID:   mediaID,
		SourceKey: out.SourceKey,
		SizeBytes: sizePtr,
		Title:     titlePtr,
	})
}

func (c *ResultsConsumer) onFetchFailed(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	hCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return c.markFail.Handle(hCtx, mediacmd.MarkFetchFailedCommand{MediaID: mediaID})
}

func (c *ResultsConsumer) onProbeSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[probeOutput](msg)
	if err != nil {
		return err
	}
	fps := 0.0
	if out.FPSDen > 0 {
		fps = float64(out.FPSNum) / float64(out.FPSDen)
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyProbe.Handle(hCtx, mediacmd.ApplyProbeResultCommand{
		MediaID:    mediaID,
		DurationMs: out.DurationMs,
		Width:      out.Width,
		Height:     out.Height,
		FPS:        fps,
		VideoCodec: out.VideoCodec,
		AudioCodec: out.AudioCodec,
		HasAudio:   out.AudioChannels > 0,
	})
}

func (c *ResultsConsumer) onAudioSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[audioOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyAudio.Handle(hCtx, mediacmd.ApplyAudioResultCommand{
		MediaID:          mediaID,
		AnalysisKey:      out.AnalysisKey,
		TranscriptionKey: out.TranscriptionKey,
		MasterKey:        out.MasterKey,
	})
}

func (c *ResultsConsumer) onProxySucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[proxyOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyProxy.Handle(hCtx, mediacmd.ApplyProxyResultCommand{
		MediaID:  mediaID,
		ProxyKey: out.ProxyKey,
	})
}

func (c *ResultsConsumer) onWaveformSucceeded(ctx context.Context, msg *envelope.Message) error {
	mediaID, err := uuid.Parse(msg.CorrelationID)
	if err != nil {
		return fmt.Errorf("invalid correlationId: %w", err)
	}
	out, err := decodeResult[waveformOutput](msg)
	if err != nil {
		return err
	}
	hCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return c.applyWave.Handle(hCtx, mediacmd.ApplyWaveformResultCommand{
		MediaID:  mediaID,
		PeaksKey: out.PeaksKey,
	})
}

func decodeResult[T any](msg *envelope.Message) (T, error) {
	var zero T
	var rp envelope.ResultPayload
	if err := json.Unmarshal(msg.Payload, &rp); err != nil {
		return zero, err
	}
	if len(rp.Output) == 0 {
		return zero, nil
	}
	var out T
	if err := json.Unmarshal(rp.Output, &out); err != nil {
		return zero, err
	}
	return out, nil
}

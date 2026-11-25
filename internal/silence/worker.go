package silence

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	s3store "go-api/internal/infrastructure/storage/s3"
	"go-api/internal/pipeline"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Config struct {
	RabbitURL   string
	MediaBucket string
	WorkDir     string
	S3          s3store.Config
	Prefetch    int
}

type jobPayload struct {
	MediaID            string   `json:"mediaId"`
	AudioKey           string   `json:"audioKey"`
	ThresholdMode      string   `json:"thresholdMode"`
	SilenceThresholdDB *float64 `json:"silenceThresholdDb"`
	NoiseFloorOffsetDb float64  `json:"noiseFloorOffsetDb"`
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		RabbitURL:   os.Getenv("RABBITMQ_URL"),
		MediaBucket: envOr("S3_BUCKET_MEDIA", "media"),
		WorkDir:     envOr("WORK_DIR", "/tmp/work"),
		S3: s3store.Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Region:    envOr("S3_REGION", "us-east-1"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
		},
		Prefetch: envInt("SILENCE_PREFETCH", 2),
	}
	if cfg.RabbitURL == "" {
		return cfg, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.S3.Endpoint == "" || cfg.S3.AccessKey == "" || cfg.S3.SecretKey == "" {
		return cfg, fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	return cfg, nil
}

func Run(ctx context.Context, cfg Config) error {
	store, err := s3store.New(cfg.S3)
	if err != nil {
		return err
	}
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()

	setup, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := topology.DeclareExchanges(setup); err != nil {
		_ = setup.Close()
		return err
	}
	if err := topology.DeclareJobQueue(setup, "silence", 30000); err != nil {
		_ = setup.Close()
		return err
	}
	_ = setup.Close()

	prefetch := cfg.Prefetch
	if prefetch <= 0 {
		prefetch = 2
	}
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	bus, err := topology.NewBus(ch, prefetch)
	if err != nil {
		return err
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume("q.silence", "silence", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf("worker-silence started (prefetch=%d)", prefetch)
	sem := make(chan struct{}, prefetch)
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case d, ok := <-deliveries:
			if !ok {
				wg.Wait()
				return fmt.Errorf("q.silence closed")
			}
			sem <- struct{}{}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer wg.Done()
				defer func() { <-sem }()
				if err := handle(ctx, bus, store, cfg, d); err != nil {
					log.Printf("worker-silence error: %v", err)
				}
			}(d)
		}
	}
}

func handle(ctx context.Context, bus *topology.Bus, store *s3store.Client, cfg Config, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	var req jobPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil || req.MediaID == "" || req.AudioKey == "" {
		return finish(ctx, bus, d, msg, false, nil, "INVALID_PAYLOAD", "mediaId and audioKey required", false)
	}
	if req.ThresholdMode == "" {
		req.ThresholdMode = "auto"
	}
	if req.NoiseFloorOffsetDb == 0 {
		req.NoiseFloorOffsetDb = 9 // standard auto offset above noise floor
	}

	workDir := filepath.Join(cfg.WorkDir, "silence-"+req.MediaID+"-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "WORKDIR", err.Error(), true)
	}
	defer os.RemoveAll(workDir)

	local := filepath.Join(workDir, "analysis.wav")
	if err := store.DownloadFile(ctx, cfg.MediaBucket, req.AudioKey, local); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "DOWNLOAD", err.Error(), true)
	}

	_ = publishProgress(ctx, bus, msg, 10, "detect", "Measuring noise floor")
	result, err := pipeline.DetectSilence(ctx, local, req.ThresholdMode, req.SilenceThresholdDB, req.NoiseFloorOffsetDb)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "SILENCE", err.Error(), false)
	}

	detectionsKey := fmt.Sprintf("media/%s/detections/silence.json", req.MediaID)
	raw, err := json.Marshal(result)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "ENCODE", err.Error(), false)
	}
	tmp := filepath.Join(workDir, "silence.json")
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "WRITE", err.Error(), true)
	}
	_ = publishProgress(ctx, bus, msg, 80, "upload", "Uploading silence detections")
	if err := store.UploadFile(ctx, cfg.MediaBucket, detectionsKey, tmp, "application/json"); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "UPLOAD", err.Error(), true)
	}

	_ = publishProgress(ctx, bus, msg, 100, "done", "Silence detection completed")
	return finish(ctx, bus, d, msg, true, map[string]any{
		"detectionsKey":         detectionsKey,
		"noiseFloorDb":          result.NoiseFloorDb,
		"calculatedThresholdDb": result.CalculatedThresholdDb,
		"intervals":             result.Intervals,
	}, "", "", false)
}

func finish(ctx context.Context, bus *topology.Bus, d amqp.Delivery, msg *envelope.Message, ok bool, output any, code, errMsg string, retryable bool) error {
	corr := msg.CorrelationID
	if corr == "" {
		corr = msg.MessageID
	}
	m, body, err := envelope.NewResult("silence", corr, msg.MessageID, msg.Attempt, ok, output, code, errMsg, retryable)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	if err := bus.PublishResult(ctx, envelope.ResultRoutingKey("silence", ok), body, m.MessageID, corr); err != nil {
		_ = bus.Nack(d)
		return err
	}
	return bus.Ack(d)
}

func publishProgress(ctx context.Context, bus *topology.Bus, msg *envelope.Message, pct float64, stage, message string) error {
	m, body, err := envelope.NewProgress("silence", msg.CorrelationID, msg.MessageID, msg.Attempt, pct, stage, message)
	if err != nil {
		return err
	}
	return bus.PublishProgress(ctx, body, m.MessageID, msg.CorrelationID)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

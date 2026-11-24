package textdetect

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
	MediaID            string `json:"mediaId"`
	WordsKey           string `json:"wordsKey"`
	Language           string `json:"language"`
	DetectionLevel     string `json:"detectionLevel"`
	FillersEnabled     bool   `json:"fillersEnabled"`
	RepetitionsEnabled bool   `json:"repetitionsEnabled"`
	FalseStartsEnabled bool   `json:"falseStartsEnabled"`
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
		Prefetch: envInt("TEXT_PREFETCH", 4),
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
	if err := topology.DeclareJobQueue(setup, "text", 30000); err != nil {
		_ = setup.Close()
		return err
	}
	_ = setup.Close()

	prefetch := cfg.Prefetch
	if prefetch <= 0 {
		prefetch = 4
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
	deliveries, err := ch.Consume("q.text", "text", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf("worker-text started (prefetch=%d)", prefetch)
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
				return fmt.Errorf("q.text closed")
			}
			sem <- struct{}{}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer wg.Done()
				defer func() { <-sem }()
				if err := handle(ctx, bus, store, cfg, d); err != nil {
					log.Printf("worker-text error: %v", err)
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
	if err := json.Unmarshal(msg.Payload, &req); err != nil || req.MediaID == "" || req.WordsKey == "" {
		return finish(ctx, bus, d, msg, false, nil, "INVALID_PAYLOAD", "mediaId and wordsKey required", false)
	}

	workDir := filepath.Join(cfg.WorkDir, "text-"+req.MediaID+"-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "WORKDIR", err.Error(), true)
	}
	defer os.RemoveAll(workDir)

	_ = publishProgress(ctx, bus, msg, 10, "download", "Downloading words.json")
	local := filepath.Join(workDir, "words.json")
	if err := store.DownloadFile(ctx, cfg.MediaBucket, req.WordsKey, local); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "DOWNLOAD", err.Error(), true)
	}
	raw, err := os.ReadFile(local)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "READ", err.Error(), true)
	}
	var payload struct {
		Words []struct {
			Text          string `json:"text"`
			SourceStartMs int64  `json:"sourceStartMs"`
			SourceEndMs   int64  `json:"sourceEndMs"`
		} `json:"words"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "PARSE", err.Error(), false)
	}
	words := make([]Word, 0, len(payload.Words))
	for i, w := range payload.Words {
		words = append(words, Word{Index: i, Text: w.Text, SourceStartMs: w.SourceStartMs, SourceEndMs: w.SourceEndMs})
	}

	_ = publishProgress(ctx, bus, msg, 50, "detect", "Running text rules")
	intervals := Detect(words, Options{
		Level:              req.DetectionLevel,
		FillersEnabled:     req.FillersEnabled,
		RepetitionsEnabled: req.RepetitionsEnabled,
		FalseStartsEnabled: req.FalseStartsEnabled,
	})

	outIntervals := make([]map[string]any, 0, len(intervals))
	for _, iv := range intervals {
		outIntervals = append(outIntervals, map[string]any{
			"kind":          iv.Kind,
			"sourceStartMs": iv.SourceStartMs,
			"sourceEndMs":   iv.SourceEndMs,
			"firstWord":     iv.FirstWord,
			"lastWord":      iv.LastWord,
		})
	}
	detectionsKey := fmt.Sprintf("media/%s/detections/text.json", req.MediaID)
	encoded, err := json.Marshal(map[string]any{
		"detector":  DetectorVersion,
		"intervals": outIntervals,
	})
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "ENCODE", err.Error(), false)
	}
	tmp := filepath.Join(workDir, "text.json")
	if err := os.WriteFile(tmp, encoded, 0o644); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "WRITE", err.Error(), true)
	}
	_ = publishProgress(ctx, bus, msg, 90, "upload", "Uploading text detections")
	if err := store.UploadFile(ctx, cfg.MediaBucket, detectionsKey, tmp, "application/json"); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "UPLOAD", err.Error(), true)
	}

	_ = publishProgress(ctx, bus, msg, 100, "done", "Text detection completed")
	return finish(ctx, bus, d, msg, true, map[string]any{
		"detectionsKey": detectionsKey,
		"detector":      DetectorVersion,
		"intervals":     outIntervals,
	}, "", "", false)
}

func finish(ctx context.Context, bus *topology.Bus, d amqp.Delivery, msg *envelope.Message, ok bool, output any, code, errMsg string, retryable bool) error {
	corr := msg.CorrelationID
	if corr == "" {
		corr = msg.MessageID
	}
	m, body, err := envelope.NewResult("text", corr, msg.MessageID, msg.Attempt, ok, output, code, errMsg, retryable)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	if err := bus.PublishResult(ctx, envelope.ResultRoutingKey("text", ok), body, m.MessageID, corr); err != nil {
		_ = bus.Nack(d)
		return err
	}
	return bus.Ack(d)
}

func publishProgress(ctx context.Context, bus *topology.Bus, msg *envelope.Message, pct float64, stage, message string) error {
	m, body, err := envelope.NewProgress("text", msg.CorrelationID, msg.MessageID, msg.Attempt, pct, stage, message)
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

package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	s3store "go-api/internal/infrastructure/storage/s3"
	"go-api/internal/pipeline"
	"go-api/internal/studio/envelope"
	"go-api/internal/studio/topology"

	amqp "github.com/rabbitmq/amqp091-go"
)

const jobType = "fetch"

type Config struct {
	RabbitURL   string
	MediaBucket string
	WorkDir     string
	Concurrency int
	S3          s3store.Config
	YTDLP       pipeline.YTDLPConfig
}

type FetchPayload struct {
	MediaID        string `json:"mediaId"`
	URL            string `json:"url"`
	MaxSizeBytes   int64  `json:"maxSizeBytes,omitempty"`
	MaxDurationSec int    `json:"maxDurationSec,omitempty"`
}

type FetchResult struct {
	SourceKey string           `json:"sourceKey"`
	Title     string           `json:"title,omitempty"`
	SizeBytes int64            `json:"sizeBytes"`
	SHA256    string           `json:"sha256"`
	Origin    *pipeline.Origin `json:"origin,omitempty"`
}

func LoadConfigFromEnv() (Config, error) {
	concurrency := envInt("FETCH_CONCURRENCY", 2)
	cfg := Config{
		RabbitURL:   os.Getenv("RABBITMQ_URL"),
		MediaBucket: envOr("S3_BUCKET_MEDIA", "media"),
		WorkDir:     envOr("WORK_DIR", "/tmp/work"),
		Concurrency: concurrency,
		S3: s3store.Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Region:    envOr("S3_REGION", "us-east-1"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
		},
		YTDLP: pipeline.YTDLPConfig{
			Bin:           envOr("YTDLP_BIN", "yt-dlp"),
			MaxFilesizeMB: envInt("YTDLP_MAX_FILESIZE_MB", 8000),
			MaxDurationS:  envInt("YTDLP_MAX_DURATION_S", 14400),
			MaxHeight:     envInt("YTDLP_MAX_HEIGHT", 720),
			CookiesFile:   os.Getenv("YTDLP_COOKIES_FILE"),
		},
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

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	defer ch.Close()

	if err := topology.DeclareExchanges(ch); err != nil {
		return err
	}
	if err := topology.DeclareJobQueue(ch, jobType, 30000); err != nil {
		return err
	}

	bus, err := topology.NewBus(ch, cfg.Concurrency)
	if err != nil {
		return err
	}
	if err := ch.Qos(cfg.Concurrency, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	queue := "q." + jobType
	deliveries, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", queue, err)
	}

	log.Printf("worker-fetch started, consuming %s (concurrency=%d)", queue, cfg.Concurrency)

	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker-fetch shutting down")
			wg.Wait()
			return nil
		case d, ok := <-deliveries:
			if !ok {
				wg.Wait()
				return fmt.Errorf("worker-fetch: deliveries closed")
			}
			select {
			case <-ctx.Done():
				wg.Wait()
				return nil
			case sem <- struct{}{}:
			}
			wg.Add(1)
			go func(d amqp.Delivery) {
				defer wg.Done()
				defer func() { <-sem }()
				if err := handle(ctx, bus, store, cfg, d); err != nil {
					log.Printf("worker-fetch handle error: %v", err)
				}
			}(d)
		}
	}
}

func handle(ctx context.Context, bus *topology.Bus, store *s3store.Client, cfg Config, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		log.Printf("worker-fetch poison: %v", err)
		return bus.Nack(d)
	}

	var req FetchPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil || req.MediaID == "" || req.URL == "" {
		return finish(ctx, bus, d, msg, jobType, false, nil, "INVALID_PAYLOAD", "mediaId and url are required", false)
	}

	ytCfg := cfg.YTDLP
	if req.MaxSizeBytes > 0 {
		ytCfg.MaxFilesizeMB = int(req.MaxSizeBytes / (1024 * 1024))
		if ytCfg.MaxFilesizeMB <= 0 {
			ytCfg.MaxFilesizeMB = 1
		}
	}
	if req.MaxDurationSec > 0 {
		ytCfg.MaxDurationS = req.MaxDurationSec
	}

	workDir := filepath.Join(cfg.WorkDir, "fetch-"+req.MediaID+"-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return finish(ctx, bus, d, msg, jobType, false, nil, "WORKDIR", err.Error(), true)
	}
	defer os.RemoveAll(workDir)

	_ = publishProgress(ctx, bus, msg, jobType, 5, "download", "Fetching source")

	localPath, origin, err := pipeline.DownloadURL(ctx, ytCfg, req.URL, workDir)
	if err != nil {
		ce := pipeline.Classify(err)
		return finish(ctx, bus, d, msg, jobType, false, nil, ce.Code, ce.Message, ce.Class == pipeline.ErrorRetryable)
	}

	ext := strings.TrimPrefix(filepath.Ext(localPath), ".")
	if ext == "" {
		ext = "mp4"
	}
	sourceKey := fmt.Sprintf("media/%s/source/original.%s", req.MediaID, ext)
	sha, size, err := pipeline.FileSHA256(localPath)
	if err != nil {
		return finish(ctx, bus, d, msg, jobType, false, nil, "HASH", err.Error(), true)
	}

	_ = publishProgress(ctx, bus, msg, jobType, 80, "upload", "Uploading source")
	if err := store.UploadFile(ctx, cfg.MediaBucket, sourceKey, localPath, pipeline.ContentTypeForExt(ext)); err != nil {
		return finish(ctx, bus, d, msg, jobType, false, nil, "UPLOAD", err.Error(), true)
	}

	out := FetchResult{
		SourceKey: sourceKey,
		SizeBytes: size,
		SHA256:    sha,
		Origin:    origin,
	}
	if origin != nil {
		out.Title = origin.Title
	}
	_ = publishProgress(ctx, bus, msg, jobType, 100, "done", "Fetch completed")
	return finish(ctx, bus, d, msg, jobType, true, out, "", "", false)
}

func finish(ctx context.Context, bus *topology.Bus, d amqp.Delivery, msg *envelope.Message, job string, ok bool, output any, code, errMsg string, retryable bool) error {
	m, body, err := envelope.NewResult(job, msg.CorrelationID, msg.MessageID, msg.Attempt, ok, output, code, errMsg, retryable)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	rk := envelope.ResultRoutingKey(job, ok)
	if err := bus.PublishResult(ctx, rk, body, m.MessageID, msg.CorrelationID); err != nil {
		_ = bus.Nack(d)
		return err
	}
	return bus.Ack(d)
}

func publishProgress(ctx context.Context, bus *topology.Bus, msg *envelope.Message, job string, pct float64, stage, message string) error {
	m, body, err := envelope.NewProgress(job, msg.CorrelationID, msg.MessageID, msg.Attempt, pct, stage, message)
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

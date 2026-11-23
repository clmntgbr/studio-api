package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	s3store "go-api/internal/infrastructure/storage/s3"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName       = "q.ingest"
	resultsExchange = "results"
)

type WorkerConfig struct {
	RabbitURL   string
	MediaBucket string
	WorkDir     string
	S3          s3store.Config
	YTDLP       YTDLPConfig
}

func LoadWorkerConfigFromEnv() (WorkerConfig, error) {
	cfg := WorkerConfig{
		RabbitURL:   os.Getenv("RABBITMQ_URL"),
		MediaBucket: envOr("S3_BUCKET_MEDIA", "media"),
		WorkDir:     envOr("WORK_DIR", "/tmp/work"),
		S3: s3store.Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Region:    envOr("S3_REGION", "us-east-1"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
		},
		YTDLP: YTDLPConfig{
			Bin:           envOr("YTDLP_BIN", "yt-dlp"),
			MaxFilesizeMB: envIntOr("YTDLP_MAX_FILESIZE_MB", 8000),
			MaxDurationS:  envIntOr("YTDLP_MAX_DURATION_S", 14400),
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

func RunWorker(ctx context.Context, cfg WorkerConfig) error {
	store, err := s3store.New(cfg.S3)
	if err != nil {
		return err
	}
	pipeline := NewPipeline(PipelineConfig{
		MediaBucket: cfg.MediaBucket,
		WorkDirRoot: cfg.WorkDir,
		YTDLP:       cfg.YTDLP,
		Storage:     store,
	})

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

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare %s: %w", queueName, err)
	}
	if err := ch.ExchangeDeclare(resultsExchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", resultsExchange, err)
	}
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("confirm mode: %w", err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume %s: %w", queueName, err)
	}

	log.Printf("worker-ingest started, consuming %s", queueName)

	for {
		select {
		case <-ctx.Done():
			log.Printf("worker-ingest shutting down")
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("worker-ingest: deliveries channel closed")
			}
			if err := handleDelivery(ctx, ch, confirms, pipeline, d); err != nil {
				log.Printf("worker-ingest handle error: %v", err)
			}
		}
	}
}

func handleDelivery(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, pipeline *Pipeline, d amqp.Delivery) error {
	env, err := ParseEnvelope(d.Body)
	if err != nil {
		log.Printf("poison message: %v", err)
		// No requeue — avoid tight loops (architecture: never Nack(requeue=true)).
		_ = d.Nack(false, false)
		return err
	}

	req, err := ParseRequest(env.Payload)
	if err != nil {
		classified := Permanent("INVALID_PAYLOAD", err.Error())
		if pubErr := publishResult(ctx, ch, confirms, env, env.MediaID, nil, classified, 0); pubErr != nil {
			log.Printf("failed to publish permanent failure: %v", pubErr)
			_ = d.Nack(false, false)
			return pubErr
		}
		return d.Ack(false)
	}

	mediaID := env.MediaID
	if mediaID == "" {
		return publishAndAckPermanent(ctx, ch, confirms, d, env, Permanent("MISSING_MEDIA_ID", "media_id is required"))
	}

	_ = publishProgress(ctx, ch, confirms, env, 0, "start", "ingest started")

	start := time.Now()
	lastProgress := time.Time{}
	completed, runErr := pipeline.Run(ctx, mediaID, req, func(pct float64, stage, message string) {
		if time.Since(lastProgress) < time.Second && pct < 100 {
			return
		}
		lastProgress = time.Now()
		_ = publishProgress(ctx, ch, confirms, env, pct, stage, message)
	})
	elapsed := time.Since(start).Milliseconds()

	if pubErr := publishResult(ctx, ch, confirms, env, mediaID, completed, runErr, elapsed); pubErr != nil {
		log.Printf("result publish failed: %v", pubErr)
		_ = d.Nack(false, false)
		return pubErr
	}
	return d.Ack(false)
}

func publishAndAckPermanent(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, d amqp.Delivery, env *Envelope, err error) error {
	if pubErr := publishResult(ctx, ch, confirms, env, env.MediaID, nil, err, 0); pubErr != nil {
		_ = d.Nack(false, false)
		return pubErr
	}
	return d.Ack(false)
}

func publishProgress(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, env *Envelope, pct float64, stage, message string) error {
	payload, err := json.Marshal(StepProgress{
		StepID:  env.StepID,
		Pct:     pct,
		Stage:   stage,
		Message: message,
	})
	if err != nil {
		return err
	}
	eventID := uuid.NewString()
	body, err := json.Marshal(Envelope{
		Schema:     SchemaStepProgress,
		EventID:    eventID,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		JobID:      env.JobID,
		StepID:     env.StepID,
		MediaID:    env.MediaID,
		Attempt:    env.Attempt,
		Payload:    payload,
	})
	if err != nil {
		return err
	}
	return publishConfirm(ctx, ch, confirms, "progress.ingest", amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Transient,
		Type:          SchemaStepProgress,
		MessageId:     eventID,
		CorrelationId: env.JobID,
		Headers:       amqp.Table{"causation_id": env.EventID},
		Body:          body,
	})
}

func publishResult(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, env *Envelope, mediaID string, out *Completed, runErr error, durationMs int64) error {
	result := StepResult{
		StepID:  env.StepID,
		Metrics: &StepMetrics{DurationMs: durationMs},
	}
	routing := "result.ingest.succeeded"
	if runErr != nil {
		routing = "result.ingest.failed"
		result.Status = "failed"
		ce := Classify(runErr)
		result.Error = &StepError{
			Code:      ce.Code,
			Message:   ce.Message,
			Retryable: ce.Class == ErrorRetryable,
		}
	} else {
		result.Status = "succeeded"
		result.Output = out
		result.OutputSchema = SchemaIngestDone
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	eventID := uuid.NewString()
	body, err := json.Marshal(Envelope{
		Schema:     SchemaStepResult,
		EventID:    eventID,
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		JobID:      env.JobID,
		StepID:     env.StepID,
		MediaID:    mediaID,
		Attempt:    env.Attempt,
		Payload:    payload,
	})
	if err != nil {
		return err
	}
	return publishConfirm(ctx, ch, confirms, routing, amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		Type:          SchemaStepResult,
		MessageId:     eventID,
		CorrelationId: env.JobID,
		Headers:       amqp.Table{"causation_id": env.EventID},
		Body:          body,
	})
}

func publishConfirm(ctx context.Context, ch *amqp.Channel, confirms <-chan amqp.Confirmation, routingKey string, msg amqp.Publishing) error {
	if err := ch.PublishWithContext(ctx, resultsExchange, routingKey, false, false, msg); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case c, ok := <-confirms:
		if !ok {
			return fmt.Errorf("confirm channel closed")
		}
		if !c.Ack {
			return fmt.Errorf("publish not confirmed for %s", routingKey)
		}
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("publish confirm timeout for %s", routingKey)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

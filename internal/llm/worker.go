package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	Provider    string
	APIKey      string
	BaseURL     string
	Model       string
}

type jobPayload struct {
	MediaID  string   `json:"mediaId"`
	WordsKey string   `json:"wordsKey"`
	Language string   `json:"language"`
	Tasks    []string `json:"tasks"`
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		RabbitURL:   os.Getenv("RABBITMQ_URL"),
		MediaBucket: envOr("S3_BUCKET_MEDIA", "media"),
		WorkDir:     envOr("WORK_DIR", "/tmp/work"),
		Provider:    strings.ToLower(envOr("LLM_PROVIDER", "mock")),
		APIKey:      os.Getenv("LLM_API_KEY"),
		BaseURL:     os.Getenv("LLM_BASE_URL"),
		Model:       envOr("LLM_MODEL", "gpt-4o-mini"),
		S3: s3store.Config{
			Endpoint:  os.Getenv("S3_ENDPOINT"),
			Region:    envOr("S3_REGION", "us-east-1"),
			AccessKey: os.Getenv("S3_ACCESS_KEY"),
			SecretKey: os.Getenv("S3_SECRET_KEY"),
		},
		Prefetch: envInt("LLM_PREFETCH", 4),
	}
	if cfg.RabbitURL == "" {
		return cfg, fmt.Errorf("RABBITMQ_URL is required")
	}
	if cfg.S3.Endpoint == "" || cfg.S3.AccessKey == "" || cfg.S3.SecretKey == "" {
		return cfg, fmt.Errorf("S3_ENDPOINT, S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	return cfg, nil
}

func NewAnalyzer(cfg Config) Analyzer {
	switch cfg.Provider {
	case "openai", "deepseek", "compatible", "grok", "xai":
		baseURL, model, provider := resolveCompatibleDefaults(cfg)
		return NewOpenAICompatibleAnalyzer(baseURL, cfg.APIKey, model, provider)
	default:
		return NewFileMockAnalyzer("")
	}
}

func resolveCompatibleDefaults(cfg Config) (baseURL, model, provider string) {
	baseURL = cfg.BaseURL
	model = cfg.Model
	provider = cfg.Provider
	switch cfg.Provider {
	case "grok", "xai":
		provider = "grok"
		if baseURL == "" {
			baseURL = "https://api.x.ai/v1"
		}
		if model == "" || model == "gpt-4o-mini" {
			model = "grok-3-mini"
		}
	case "deepseek":
		if baseURL == "" {
			baseURL = "https://api.deepseek.com/v1"
		}
		if model == "" || model == "gpt-4o-mini" {
			model = "deepseek-chat"
		}
	case "openai", "compatible":
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		if model == "" {
			model = "gpt-4o-mini"
		}
	}
	return baseURL, model, provider
}

func Run(ctx context.Context, cfg Config) error {
	store, err := s3store.New(cfg.S3)
	if err != nil {
		return err
	}
	analyzer := NewAnalyzer(cfg)

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
	if err := topology.DeclareJobQueue(setup, "llm", 300000); err != nil {
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
	deliveries, err := ch.Consume("q.llm", "worker-llm", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Printf("worker-llm started provider=%s", cfg.Provider)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("deliveries closed")
			}
			if err := handle(ctx, cfg, store, analyzer, bus, d); err != nil {
				log.Printf("worker-llm error: %v", err)
			}
		}
	}
}

func handle(ctx context.Context, cfg Config, store *s3store.Client, analyzer Analyzer, bus *topology.Bus, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	var req jobPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil || req.MediaID == "" || req.WordsKey == "" {
		return finish(ctx, bus, d, msg, false, nil, "invalid_payload", "invalid llm job", false)
	}

	workDir := filepath.Join(cfg.WorkDir, "llm-"+req.MediaID)
	_ = os.RemoveAll(workDir)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "workdir", err.Error(), true)
	}
	defer os.RemoveAll(workDir)

	_ = publishProgress(ctx, bus, msg, 10, "download", "Downloading words.json")
	local := filepath.Join(workDir, "words.json")
	if err := store.DownloadFile(ctx, cfg.MediaBucket, req.WordsKey, local); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "download", err.Error(), true)
	}
	raw, err := os.ReadFile(local)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "read", err.Error(), false)
	}
	words, err := parseWords(raw)
	if err != nil {
		return finish(ctx, bus, d, msg, false, nil, "words", err.Error(), false)
	}

	_ = publishProgress(ctx, bus, msg, 40, "analyze", "Running LLM analysis")
	result, err := analyzer.Analyze(ctx, words, req.Tasks, req.Language)
	if err != nil {
		// One retry for invalid / transient provider errors.
		result, err = analyzer.Analyze(ctx, words, req.Tasks, req.Language)
		if err != nil {
			return finish(ctx, bus, d, msg, false, nil, "analyze", err.Error(), true)
		}
	}

	artifactsKey := fmt.Sprintf("media/%s/analysis/llm.json", req.MediaID)
	artifactBody, _ := json.Marshal(result)
	tmp := filepath.Join(workDir, "llm.json")
	if err := os.WriteFile(tmp, artifactBody, 0o644); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "write", err.Error(), false)
	}
	_ = publishProgress(ctx, bus, msg, 85, "upload", "Uploading llm.json")
	if err := store.UploadFile(ctx, cfg.MediaBucket, artifactsKey, tmp, "application/json"); err != nil {
		return finish(ctx, bus, d, msg, false, nil, "upload", err.Error(), true)
	}
	_ = publishProgress(ctx, bus, msg, 100, "done", "LLM analysis completed")
	return finish(ctx, bus, d, msg, true, map[string]any{
		"artifactsKey": artifactsKey,
		"provider":     result.Provider,
		"sponsors":     result.Sponsors,
		"chapters":     result.Chapters,
		"clips":        result.Clips,
	}, "", "", false)
}

func parseWords(raw []byte) ([]Word, error) {
	var asArray []Word
	if err := json.Unmarshal(raw, &asArray); err == nil {
		return asArray, nil
	}
	var wrap struct {
		Words []Word `json:"words"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	return wrap.Words, nil
}

func finish(ctx context.Context, bus *topology.Bus, d amqp.Delivery, msg *envelope.Message, ok bool, output any, code, errMsg string, retryable bool) error {
	corr := msg.CorrelationID
	if corr == "" {
		corr = msg.MessageID
	}
	m, body, err := envelope.NewResult("llm", corr, msg.MessageID, msg.Attempt, ok, output, code, errMsg, retryable)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	if err := bus.PublishResult(ctx, envelope.ResultRoutingKey("llm", ok), body, m.MessageID, corr); err != nil {
		_ = bus.Nack(d)
		return err
	}
	return bus.Ack(d)
}

func publishProgress(ctx context.Context, bus *topology.Bus, msg *envelope.Message, pct float64, stage, message string) error {
	m, body, err := envelope.NewProgress("llm", msg.CorrelationID, msg.MessageID, msg.Attempt, pct, stage, message)
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

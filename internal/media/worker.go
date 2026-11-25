package media

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

type Config struct {
	RabbitURL   string
	MediaBucket string
	WorkDir     string
	S3          s3store.Config
	Prefetch    map[string]int
}

type sourcePayload struct {
	MediaID   string `json:"mediaId"`
	SourceKey string `json:"sourceKey"`
}

type waveformPayload struct {
	MediaID  string `json:"mediaId"`
	AudioKey string `json:"audioKey"`
}

type proxyPayload struct {
	MediaID   string             `json:"mediaId"`
	SourceKey string             `json:"sourceKey"`
	Proxy     pipeline.ProxySpec `json:"proxy"`
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
		Prefetch: map[string]int{
			"probe":    envInt("MEDIA_PREFETCH_PROBE", 4),
			"audio":    envInt("MEDIA_PREFETCH_AUDIO", 4),
			"proxy":    envInt("MEDIA_PREFETCH_PROXY", 1),
			"waveform": envInt("MEDIA_PREFETCH_WAVEFORM", 4),
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

	setup, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := topology.DeclareExchanges(setup); err != nil {
		_ = setup.Close()
		return err
	}
	for _, job := range []string{"probe", "audio", "proxy", "waveform"} {
		if err := topology.DeclareJobQueue(setup, job, 30000); err != nil {
			_ = setup.Close()
			return err
		}
	}
	_ = setup.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, 4)

	start := func(job string, handler func(context.Context, *topology.Bus, *s3store.Client, Config, amqp.Delivery) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := consumeJob(ctx, conn, store, cfg, job, cfg.Prefetch[job], handler); err != nil && ctx.Err() == nil {
				errCh <- err
			}
		}()
	}

	start("probe", handleProbe)
	start("audio", handleAudio)
	start("proxy", handleProxy)
	start("waveform", handleWaveform)

	log.Printf("worker-media started (probe=%d audio=%d proxy=%d waveform=%d)",
		cfg.Prefetch["probe"], cfg.Prefetch["audio"], cfg.Prefetch["proxy"], cfg.Prefetch["waveform"])

	select {
	case <-ctx.Done():
		wg.Wait()
		return nil
	case err := <-errCh:
		wg.Wait()
		return err
	}
}

func consumeJob(
	ctx context.Context,
	conn *amqp.Connection,
	store *s3store.Client,
	cfg Config,
	job string,
	prefetch int,
	handler func(context.Context, *topology.Bus, *s3store.Client, Config, amqp.Delivery) error,
) error {
	if prefetch <= 0 {
		prefetch = 1
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

	queue := "q." + job
	deliveries, err := ch.Consume(queue, "media-"+job, false, false, false, false, nil)
	if err != nil {
		return err
	}

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
				return fmt.Errorf("worker-media %s: deliveries closed", job)
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
				if err := handler(ctx, bus, store, cfg, d); err != nil {
					log.Printf("worker-media %s error: %v", job, err)
				}
			}(d)
		}
	}
}

func handleProbe(ctx context.Context, bus *topology.Bus, store *s3store.Client, cfg Config, d amqp.Delivery) error {
	msg, req, err := parseSourceJob(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	local, cleanup, err := downloadToWork(ctx, store, cfg, req.MediaID, req.SourceKey, "probe")
	if err != nil {
		return finish(ctx, bus, d, msg, "probe", false, nil, "DOWNLOAD", err.Error(), true)
	}
	defer cleanup()

	probe, err := pipeline.ProbeFile(ctx, local)
	if err != nil {
		return finish(ctx, bus, d, msg, "probe", false, nil, "PROBE", err.Error(), false)
	}
	return finish(ctx, bus, d, msg, "probe", true, probe, "", "", false)
}

func handleAudio(ctx context.Context, bus *topology.Bus, store *s3store.Client, cfg Config, d amqp.Delivery) error {
	msg, req, err := parseSourceJob(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	_ = publishProgress(ctx, bus, msg, "audio", 1, "download", "Downloading source for audio")
	local, cleanup, err := downloadToWork(ctx, store, cfg, req.MediaID, req.SourceKey, "audio")
	if err != nil {
		return finish(ctx, bus, d, msg, "audio", false, nil, "DOWNLOAD", err.Error(), true)
	}
	defer cleanup()

	workDir := filepath.Dir(local)
	analysisPath := filepath.Join(workDir, "analysis_16k_mono.wav")
	txPath := filepath.Join(workDir, "transcription.m4a")
	masterPath := filepath.Join(workDir, "master.flac")

	_ = publishProgress(ctx, bus, msg, "audio", 20, "extract", "Extracting analysis / transcription / master")
	type extractErr struct {
		code string
		err  error
	}
	errCh := make(chan extractErr, 3)
	var extractWG sync.WaitGroup
	extractWG.Add(3)
	go func() {
		defer extractWG.Done()
		if err := pipeline.ExtractAnalysisAudio(ctx, local, analysisPath, pipeline.AudioAnalysisSpec{}); err != nil {
			errCh <- extractErr{"AUDIO_ANALYSIS", err}
		}
	}()
	go func() {
		defer extractWG.Done()
		if err := pipeline.ExtractTranscriptionAudio(ctx, local, txPath, pipeline.AudioTranscriptionSpec{}); err != nil {
			errCh <- extractErr{"AUDIO_TRANSCRIPTION", err}
		}
	}()
	go func() {
		defer extractWG.Done()
		if err := pipeline.ExtractMasterAudio(ctx, local, masterPath, pipeline.AudioMasterSpec{}); err != nil {
			errCh <- extractErr{"AUDIO_MASTER", err}
		}
	}()
	extractWG.Wait()
	close(errCh)
	if first, ok := <-errCh; ok {
		return finish(ctx, bus, d, msg, "audio", false, nil, first.code, first.err.Error(), false)
	}

	analysisKey := fmt.Sprintf("media/%s/audio/analysis_16k_mono.wav", req.MediaID)
	txKey := fmt.Sprintf("media/%s/audio/transcription.m4a", req.MediaID)
	masterKey := fmt.Sprintf("media/%s/audio/master.flac", req.MediaID)

	_ = publishProgress(ctx, bus, msg, "audio", 80, "upload", "Uploading audio derivatives")
	type uploadErr struct {
		err error
	}
	upCh := make(chan uploadErr, 3)
	var upWG sync.WaitGroup
	upWG.Add(3)
	go func() {
		defer upWG.Done()
		if err := store.UploadFile(ctx, cfg.MediaBucket, analysisKey, analysisPath, "audio/wav"); err != nil {
			upCh <- uploadErr{err}
		}
	}()
	go func() {
		defer upWG.Done()
		if err := store.UploadFile(ctx, cfg.MediaBucket, txKey, txPath, "audio/mp4"); err != nil {
			upCh <- uploadErr{err}
		}
	}()
	go func() {
		defer upWG.Done()
		if err := store.UploadFile(ctx, cfg.MediaBucket, masterKey, masterPath, "audio/flac"); err != nil {
			upCh <- uploadErr{err}
		}
	}()
	upWG.Wait()
	close(upCh)
	if first, ok := <-upCh; ok {
		return finish(ctx, bus, d, msg, "audio", false, nil, "UPLOAD", first.err.Error(), true)
	}
	sha, _, err := pipeline.FileSHA256(analysisPath)
	if err != nil {
		return finish(ctx, bus, d, msg, "audio", false, nil, "HASH", err.Error(), true)
	}

	return finish(ctx, bus, d, msg, "audio", true, map[string]string{
		"analysisKey":      analysisKey,
		"transcriptionKey": txKey,
		"masterKey":        masterKey,
		"sha256":           sha,
	}, "", "", false)
}

func handleProxy(ctx context.Context, bus *topology.Bus, store *s3store.Client, cfg Config, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	var req proxyPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil || req.MediaID == "" || req.SourceKey == "" {
		return finish(ctx, bus, d, msg, "proxy", false, nil, "INVALID_PAYLOAD", "mediaId and sourceKey required", false)
	}

	_ = publishProgress(ctx, bus, msg, "proxy", 1, "download", "Downloading source for proxy")
	local, cleanup, err := downloadToWork(ctx, store, cfg, req.MediaID, req.SourceKey, "proxy")
	if err != nil {
		return finish(ctx, bus, d, msg, "proxy", false, nil, "DOWNLOAD", err.Error(), true)
	}
	defer cleanup()

	_ = publishProgress(ctx, bus, msg, "proxy", 15, "ffmpeg", "Encoding 720p proxy")
	dest := filepath.Join(filepath.Dir(local), "proxy_720p.mp4")
	if err := pipeline.MakeProxy(ctx, local, dest, req.Proxy); err != nil {
		return finish(ctx, bus, d, msg, "proxy", false, nil, "PROXY", err.Error(), false)
	}
	_ = publishProgress(ctx, bus, msg, "proxy", 90, "upload", "Uploading proxy")
	proxyKey := fmt.Sprintf("media/%s/proxy/proxy_720p.mp4", req.MediaID)
	if err := store.UploadFile(ctx, cfg.MediaBucket, proxyKey, dest, "video/mp4"); err != nil {
		return finish(ctx, bus, d, msg, "proxy", false, nil, "UPLOAD", err.Error(), true)
	}
	return finish(ctx, bus, d, msg, "proxy", true, map[string]string{"proxyKey": proxyKey}, "", "", false)
}

func handleWaveform(ctx context.Context, bus *topology.Bus, store *s3store.Client, cfg Config, d amqp.Delivery) error {
	msg, err := envelope.Parse(d.Body)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	var req waveformPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil || req.MediaID == "" || req.AudioKey == "" {
		return finish(ctx, bus, d, msg, "waveform", false, nil, "INVALID_PAYLOAD", "mediaId and audioKey required", false)
	}

	_ = publishProgress(ctx, bus, msg, "waveform", 1, "download", "Downloading analysis audio")
	local, cleanup, err := downloadToWork(ctx, store, cfg, req.MediaID, req.AudioKey, "waveform")
	if err != nil {
		return finish(ctx, bus, d, msg, "waveform", false, nil, "DOWNLOAD", err.Error(), true)
	}
	defer cleanup()

	_ = publishProgress(ctx, bus, msg, "waveform", 40, "peaks", "Building waveform peaks")
	peaksPath := filepath.Join(filepath.Dir(local), "peaks.bin")
	if err := pipeline.MakeWaveformPeaks(ctx, local, peaksPath); err != nil {
		return finish(ctx, bus, d, msg, "waveform", false, nil, "WAVEFORM", err.Error(), false)
	}
	peaksKey := fmt.Sprintf("media/%s/peaks/peaks.bin", req.MediaID)
	_ = publishProgress(ctx, bus, msg, "waveform", 90, "upload", "Uploading peaks")
	if err := store.UploadFile(ctx, cfg.MediaBucket, peaksKey, peaksPath, "application/octet-stream"); err != nil {
		return finish(ctx, bus, d, msg, "waveform", false, nil, "UPLOAD", err.Error(), true)
	}
	return finish(ctx, bus, d, msg, "waveform", true, map[string]string{"peaksKey": peaksKey}, "", "", false)
}

func parseSourceJob(body []byte) (*envelope.Message, sourcePayload, error) {
	msg, err := envelope.Parse(body)
	if err != nil {
		return nil, sourcePayload{}, err
	}
	var req sourcePayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return msg, sourcePayload{}, err
	}
	if req.MediaID == "" || req.SourceKey == "" {
		return msg, req, fmt.Errorf("mediaId and sourceKey required")
	}
	return msg, req, nil
}

func downloadToWork(ctx context.Context, store *s3store.Client, cfg Config, mediaID, key, kind string) (string, func(), error) {
	workDir := filepath.Join(cfg.WorkDir, kind+"-"+mediaID+"-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(workDir) }
	ext := strings.TrimPrefix(filepath.Ext(key), ".")
	if ext == "" {
		ext = "bin"
	}
	dest := filepath.Join(workDir, "input."+ext)
	if err := store.DownloadFile(ctx, cfg.MediaBucket, key, dest); err != nil {
		cleanup()
		return "", nil, err
	}
	return dest, cleanup, nil
}

func finish(ctx context.Context, bus *topology.Bus, d amqp.Delivery, msg *envelope.Message, job string, ok bool, output any, code, errMsg string, retryable bool) error {
	corr := msg.CorrelationID
	if corr == "" {
		corr = msg.MessageID
	}
	m, body, err := envelope.NewResult(job, corr, msg.MessageID, msg.Attempt, ok, output, code, errMsg, retryable)
	if err != nil {
		_ = bus.Nack(d)
		return err
	}
	if err := bus.PublishResult(ctx, envelope.ResultRoutingKey(job, ok), body, m.MessageID, corr); err != nil {
		_ = bus.Nack(d)
		return err
	}
	return bus.Ack(d)
}

func publishProgress(ctx context.Context, bus *topology.Bus, msg *envelope.Message, job string, pct float64, stage, message string) error {
	corr := msg.CorrelationID
	if corr == "" {
		corr = msg.MessageID
	}
	m, body, err := envelope.NewProgress(job, corr, msg.MessageID, msg.Attempt, pct, stage, message)
	if err != nil {
		return err
	}
	return bus.PublishProgress(ctx, body, m.MessageID, corr)
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

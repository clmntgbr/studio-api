package ingest

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	s3store "go-api/internal/infrastructure/storage/s3"
)

type ProgressFunc func(pct float64, stage, message string)

type PipelineConfig struct {
	MediaBucket string
	WorkDirRoot string
	YTDLP       YTDLPConfig
	Storage     *s3store.Client
}

type Pipeline struct {
	cfg PipelineConfig
}

func NewPipeline(cfg PipelineConfig) *Pipeline {
	if cfg.MediaBucket == "" {
		cfg.MediaBucket = "media"
	}
	if cfg.WorkDirRoot == "" {
		cfg.WorkDirRoot = "/tmp/work"
	}
	return &Pipeline{cfg: cfg}
}

func (p *Pipeline) Run(ctx context.Context, mediaID string, req *Request, onProgress ProgressFunc) (*Completed, error) {
	if mediaID == "" {
		return nil, fmt.Errorf("media_id is required")
	}
	devLogs := isDev()
	startedAt := time.Now()
	report := func(pct float64, stage, msg string) {
		if devLogs {
			log.Printf("ingest[%s] %5.1f%% %-18s %s (+%s)", mediaID, pct, stage, msg, time.Since(startedAt).Round(time.Millisecond))
		}
		if onProgress != nil {
			onProgress(pct, stage, msg)
		}
	}

	workDir := filepath.Join(p.cfg.WorkDirRoot, mediaID+"-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir work: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(workDir); err != nil {
			log.Printf("ingest: cleanup %s: %v", workDir, err)
		}
	}()

	report(2, "fetch_source", "Fetching source")
	localPath, origin, err := p.fetchSource(ctx, workDir, req)
	if err != nil {
		return nil, err
	}

	ext := strings.TrimPrefix(filepath.Ext(localPath), ".")
	if ext == "" {
		ext = "mp4"
	}
	originalKey := fmt.Sprintf("sources/%s/original.%s", mediaID, ext)

	report(12, "upload_original", "Uploading original")
	sha, size, err := FileSHA256(localPath)
	if err != nil {
		return nil, err
	}
	if err := p.cfg.Storage.UploadFile(ctx, p.cfg.MediaBucket, originalKey, localPath, contentTypeForExt(ext)); err != nil {
		return nil, err
	}

	report(20, "probe", "Probing media")
	probe, err := ProbeFile(ctx, localPath)
	if err != nil {
		return nil, err
	}
	maxDurS := p.cfg.YTDLP.MaxDurationS
	if req.Source.MaxDurationS > 0 {
		maxDurS = req.Source.MaxDurationS
	}
	if maxDurS <= 0 {
		maxDurS = 14400
	}
	if probe.DurationMs > int64(maxDurS)*1000 {
		return nil, Permanent("DURATION_LIMIT", fmt.Sprintf("duration %dms exceeds limit %ds", probe.DurationMs, maxDurS))
	}

	working := localPath
	var mezzanine *ObjectRef
	if req.NormalizeCFR && needsMezzanine(probe) {
		report(30, "mezzanine", "Creating CFR mezzanine")
		mezPath := filepath.Join(workDir, "mezzanine.mp4")
		if err := MakeMezzanine(ctx, localPath, mezPath); err != nil {
			return nil, err
		}
		mezKey := fmt.Sprintf("derived/%s/mezzanine.mp4", mediaID)
		if err := p.cfg.Storage.UploadFile(ctx, p.cfg.MediaBucket, mezKey, mezPath, "video/mp4"); err != nil {
			return nil, err
		}
		cfr := true
		mezzanine = &ObjectRef{Key: mezKey, CFR: &cfr}
		working = mezPath
		probe, err = ProbeFile(ctx, working)
		if err != nil {
			return nil, err
		}
		probe.VFR = false
	}

	report(45, "audio_analysis", "Extracting analysis audio")
	analysisPath := filepath.Join(workDir, "audio_16k_mono.wav")
	if err := ExtractAnalysisAudio(ctx, working, analysisPath, req.Audio.Analysis); err != nil {
		return nil, err
	}
	analysisKey := fmt.Sprintf("derived/%s/audio_16k_mono.wav", mediaID)
	if err := p.cfg.Storage.UploadFile(ctx, p.cfg.MediaBucket, analysisKey, analysisPath, "audio/wav"); err != nil {
		return nil, err
	}

	report(60, "audio_transcription", "Extracting transcription audio")
	txPath := filepath.Join(workDir, "audio_transcription.m4a")
	if err := ExtractTranscriptionAudio(ctx, working, txPath, req.Audio.Transcription); err != nil {
		return nil, err
	}
	txKey := fmt.Sprintf("derived/%s/audio_transcription.m4a", mediaID)
	if err := p.cfg.Storage.UploadFile(ctx, p.cfg.MediaBucket, txKey, txPath, "audio/mp4"); err != nil {
		return nil, err
	}

	report(75, "audio_master", "Extracting master audio")
	masterPath := filepath.Join(workDir, "audio_master.flac")
	if err := ExtractMasterAudio(ctx, working, masterPath, req.Audio.Master); err != nil {
		return nil, err
	}
	masterKey := fmt.Sprintf("derived/%s/audio_master.flac", mediaID)
	if err := p.cfg.Storage.UploadFile(ctx, p.cfg.MediaBucket, masterKey, masterPath, "audio/flac"); err != nil {
		return nil, err
	}
	audioSHA, _, err := FileSHA256(analysisPath)
	if err != nil {
		return nil, err
	}

	report(90, "sprite", "Building sprite")
	spritePath := filepath.Join(workDir, "sprite.jpg")
	spriteMeta, err := MakeSprite(ctx, working, spritePath, probe.DurationMs)
	if err != nil {
		return nil, err
	}
	spriteKey := fmt.Sprintf("derived/%s/sprite.jpg", mediaID)
	if err := p.cfg.Storage.UploadFile(ctx, p.cfg.MediaBucket, spriteKey, spritePath, "image/jpeg"); err != nil {
		return nil, err
	}
	spriteMeta.Key = spriteKey

	report(100, "done", "Ingest completed")
	return &Completed{
		Source: SourceOut{
			Bucket:    p.cfg.MediaBucket,
			Key:       originalKey,
			SHA256:    sha,
			SizeBytes: size,
		},
		Probe:     probe,
		Mezzanine: mezzanine,
		Audio: AudioOut{
			AnalysisKey:      analysisKey,
			TranscriptionKey: txKey,
			MasterKey:        masterKey,
			SHA256:           audioSHA,
		},
		Sprite: spriteMeta,
		Origin: origin,
	}, nil
}

func (p *Pipeline) fetchSource(ctx context.Context, workDir string, req *Request) (string, *OriginOut, error) {
	switch req.Source.Kind {
	case "url":
		maxMB := req.Source.MaxFilesizeMB
		maxDur := req.Source.MaxDurationS
		cfg := p.cfg.YTDLP
		if maxMB > 0 {
			cfg.MaxFilesizeMB = maxMB
		}
		if maxDur > 0 {
			cfg.MaxDurationS = maxDur
		}
		return DownloadURL(ctx, cfg, req.Source.URL, workDir)
	case "upload":
		ext := strings.TrimPrefix(filepath.Ext(req.Source.Key), ".")
		if ext == "" {
			ext = "bin"
		}
		dest := filepath.Join(workDir, "original."+ext)
		if err := p.cfg.Storage.DownloadFile(ctx, req.Source.Bucket, req.Source.Key, dest); err != nil {
			return "", nil, err
		}
		return dest, nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported source.kind %q", req.Source.Kind)
	}
}

func contentTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case "mp4", "m4v":
		return "video/mp4"
	case "mov":
		return "video/quicktime"
	case "webm":
		return "video/webm"
	case "mkv":
		return "video/x-matroska"
	default:
		return "application/octet-stream"
	}
}

func isDev() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GO_ENV")))
	return env == "" || env == "development" || env == "dev" || env == "local"
}

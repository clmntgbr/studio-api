package ingest

import (
	"encoding/json"
	"fmt"
)

const (
	SchemaEnvelope      = "envelope.v1"
	SchemaIngestRequest = "ingest.requested.v1"
	SchemaIngestDone    = "ingest.completed.v1"
	SchemaStepResult    = "step.result.v1"
	SchemaStepProgress  = "step.progress.v1"
)

// Envelope is the common AMQP message wrapper (features.md §8.2).
type Envelope struct {
	Schema         string          `json:"schema"`
	EventID        string          `json:"event_id"`
	OccurredAt     string          `json:"occurred_at"`
	JobID          string          `json:"job_id"`
	StepID         string          `json:"step_id"`
	MediaID        string          `json:"media_id,omitempty"`
	ClipID         *string         `json:"clip_id,omitempty"`
	Attempt        int             `json:"attempt"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	DeadlineAt     *string         `json:"deadline_at,omitempty"`
	Payload        json.RawMessage `json:"payload"`
}

type Request struct {
	Source       SourceSpec  `json:"source"`
	Proxy        ProxySpec   `json:"proxy"`
	Audio        AudioSpec   `json:"audio"`
	NormalizeCFR bool        `json:"normalize_cfr"`
}

type SourceSpec struct {
	Kind           string `json:"kind"` // upload | url
	Bucket         string `json:"bucket,omitempty"`
	Key            string `json:"key,omitempty"`
	URL            string `json:"url,omitempty"`
	MaxDurationS   int    `json:"max_duration_s,omitempty"`
	MaxFilesizeMB  int    `json:"max_filesize_mb,omitempty"`
}

type ProxySpec struct {
	Height    int `json:"height"`
	GOPFrames int `json:"gop_frames"`
	CRF       int `json:"crf"`
}

type AudioSpec struct {
	Analysis       AudioAnalysisSpec       `json:"analysis"`
	Transcription  AudioTranscriptionSpec  `json:"transcription"`
	Master         AudioMasterSpec         `json:"master"`
}

type AudioAnalysisSpec struct {
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"`
}

type AudioTranscriptionSpec struct {
	Codec       string `json:"codec"`
	BitrateKbps int    `json:"bitrate_kbps"`
}

type AudioMasterSpec struct {
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"`
}

type Completed struct {
	Source     SourceOut   `json:"source"`
	Probe      Probe       `json:"probe"`
	Mezzanine  *ObjectRef  `json:"mezzanine,omitempty"`
	Proxy      ObjectRef   `json:"proxy"`
	Audio      AudioOut    `json:"audio"`
	Sprite     SpriteOut   `json:"sprite"`
	Origin     *OriginOut  `json:"origin,omitempty"`
}

type SourceOut struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

type ObjectRef struct {
	Key string `json:"key"`
	CFR *bool  `json:"cfr,omitempty"`
}

type Probe struct {
	DurationMs    int64   `json:"duration_ms"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	FPSNum        int     `json:"fps_num"`
	FPSDen        int     `json:"fps_den"`
	VFR           bool    `json:"vfr"`
	VideoCodec    string  `json:"video_codec"`
	AudioCodec    string  `json:"audio_codec"`
	AudioChannels int     `json:"audio_channels"`
	Rotation      int     `json:"rotation"`
}

type AudioOut struct {
	AnalysisKey       string `json:"analysis_key"`
	TranscriptionKey  string `json:"transcription_key"`
	MasterKey         string `json:"master_key"`
	SHA256            string `json:"sha256"`
}

type SpriteOut struct {
	Key         string `json:"key"`
	IntervalMs  int    `json:"interval_ms"`
	TileW       int    `json:"tile_w"`
	TileH       int    `json:"tile_h"`
	Cols        int    `json:"cols"`
}

type OriginOut struct {
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Uploader  string `json:"uploader,omitempty"`
	Extractor string `json:"extractor,omitempty"`
}

type StepResult struct {
	StepID       string         `json:"step_id"`
	Status       string         `json:"status"` // succeeded | failed
	Output       *Completed     `json:"output,omitempty"`
	OutputSchema string         `json:"output_schema,omitempty"`
	Error        *StepError     `json:"error,omitempty"`
	Metrics      *StepMetrics   `json:"metrics,omitempty"`
}

type StepError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type StepMetrics struct {
	DurationMs int64 `json:"duration_ms"`
}

type StepProgress struct {
	StepID  string  `json:"step_id"`
	Pct     float64 `json:"pct"`
	Stage   string  `json:"stage,omitempty"`
	Message string  `json:"message,omitempty"`
}

func ParseEnvelope(body []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("invalid envelope json: %w", err)
	}
	if env.Schema != "" && env.Schema != SchemaEnvelope {
		return nil, fmt.Errorf("unexpected envelope schema %q", env.Schema)
	}
	if env.StepID == "" {
		return nil, fmt.Errorf("envelope.step_id is required")
	}
	if env.Attempt < 1 {
		env.Attempt = 1
	}
	return &env, nil
}

func ParseRequest(payload json.RawMessage) (*Request, error) {
	var dto struct {
		Source       SourceSpec `json:"source"`
		Proxy        ProxySpec  `json:"proxy"`
		Audio        AudioSpec  `json:"audio"`
		NormalizeCFR *bool      `json:"normalize_cfr"`
	}
	if err := json.Unmarshal(payload, &dto); err != nil {
		return nil, fmt.Errorf("invalid ingest.requested payload: %w", err)
	}
	req := Request{
		Source:       dto.Source,
		Proxy:        dto.Proxy,
		Audio:        dto.Audio,
		NormalizeCFR: true,
	}
	if dto.NormalizeCFR != nil {
		req.NormalizeCFR = *dto.NormalizeCFR
	}
	if req.Source.Kind == "" {
		return nil, fmt.Errorf("source.kind is required")
	}
	switch req.Source.Kind {
	case "url":
		if req.Source.URL == "" {
			return nil, fmt.Errorf("source.url is required for kind=url")
		}
	case "upload":
		if req.Source.Bucket == "" || req.Source.Key == "" {
			return nil, fmt.Errorf("source.bucket and source.key are required for kind=upload")
		}
	default:
		return nil, fmt.Errorf("unsupported source.kind %q", req.Source.Kind)
	}
	if req.Proxy.Height == 0 {
		req.Proxy.Height = 720
	}
	if req.Proxy.GOPFrames == 0 {
		req.Proxy.GOPFrames = 15
	}
	if req.Proxy.CRF == 0 {
		req.Proxy.CRF = 26
	}
	if req.Audio.Analysis.SampleRate == 0 {
		req.Audio.Analysis.SampleRate = 16000
	}
	if req.Audio.Analysis.Channels == 0 {
		req.Audio.Analysis.Channels = 1
	}
	if req.Audio.Analysis.Format == "" {
		req.Audio.Analysis.Format = "wav"
	}
	if req.Audio.Transcription.Codec == "" {
		req.Audio.Transcription.Codec = "aac"
	}
	if req.Audio.Transcription.BitrateKbps == 0 {
		req.Audio.Transcription.BitrateKbps = 96
	}
	if req.Audio.Master.SampleRate == 0 {
		req.Audio.Master.SampleRate = 48000
	}
	if req.Audio.Master.Channels == 0 {
		req.Audio.Master.Channels = 2
	}
	if req.Audio.Master.Format == "" {
		req.Audio.Master.Format = "flac"
	}
	return &req, nil
}

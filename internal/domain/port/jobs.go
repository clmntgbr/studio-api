package port

import (
	"context"
	"encoding/json"
)

// JobPublisher publishes pipeline jobs onto studio.jobs.
type JobPublisher interface {
	PublishFetchJob(ctx context.Context, mediaID, rawURL string) error
	PublishProbeJob(ctx context.Context, mediaID, sourceKey string) error
	PublishAudioJob(ctx context.Context, mediaID, sourceKey string) error
	PublishProxyJob(ctx context.Context, mediaID, sourceKey string) error
	PublishWaveformJob(ctx context.Context, mediaID, audioKey string) error
	PublishSilenceJob(ctx context.Context, mediaID, audioKey string, opts SilenceJobOptions) error
	PublishTranscribeJob(ctx context.Context, mediaID, audioKey string, opts TranscribeJobOptions) error
	PublishTextJob(ctx context.Context, mediaID, wordsKey string, opts TextJobOptions) error
	PublishLLMJob(ctx context.Context, mediaID, wordsKey string, opts LLMJobOptions) error
	PublishReframeJob(ctx context.Context, mediaID, proxyKey string) error
	PublishRenderJob(ctx context.Context, renderID string, plan json.RawMessage) error
}

// LLMJobOptions configures worker-llm tasks (sponsors, chapters, viral clips).
type LLMJobOptions struct {
	Language       string
	Tasks          []string // sponsors | chapters | viral
	SponsorsAction string
}

// TextJobOptions configures the text-detection worker (fillers, repetitions, false starts).
type TextJobOptions struct {
	Language           string
	DetectionLevel     string
	FillersEnabled     bool
	RepetitionsEnabled bool
	FalseStartsEnabled bool
}

// SilenceJobOptions configures the silence detection worker.
type SilenceJobOptions struct {
	ThresholdMode      string
	SilenceThresholdDb *float64
	NoiseFloorOffsetDb float64
}

// TranscribeJobOptions configures the transcription worker.
type TranscribeJobOptions struct {
	Language *string
}

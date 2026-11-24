package port

import "context"

// JobPublisher publishes pipeline jobs onto studio.jobs.
type JobPublisher interface {
	PublishFetchJob(ctx context.Context, mediaID, rawURL string) error
	PublishProbeJob(ctx context.Context, mediaID, sourceKey string) error
	PublishAudioJob(ctx context.Context, mediaID, sourceKey string) error
	PublishProxyJob(ctx context.Context, mediaID, sourceKey string) error
	PublishWaveformJob(ctx context.Context, mediaID, audioKey string) error
	PublishSilenceJob(ctx context.Context, mediaID, audioKey string, opts SilenceJobOptions) error
}

// SilenceJobOptions configures the silence detection worker.
type SilenceJobOptions struct {
	ThresholdMode      string
	SilenceThresholdDb *float64
	NoiseFloorOffsetDb float64
}

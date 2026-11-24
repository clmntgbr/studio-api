package port

import "context"

// JobPublisher publishes pipeline jobs onto studio.jobs.
type JobPublisher interface {
	PublishFetchJob(ctx context.Context, mediaID, rawURL string) error
	PublishProbeJob(ctx context.Context, mediaID, sourceKey string) error
	PublishAudioJob(ctx context.Context, mediaID, sourceKey string) error
	PublishProxyJob(ctx context.Context, mediaID, sourceKey string) error
	PublishWaveformJob(ctx context.Context, mediaID, audioKey string) error
	PublishSilenceJob(ctx context.Context, mediaID, audioKey, thresholdMode string, silenceThresholdDb *float64) error
}

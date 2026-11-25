package media

import (
	"context"
	"encoding/json"
	"fmt"

	"go-api/internal/application/messaging"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

// EnqueueProbeHandler starts the shared post-source pipeline (probe) from media.source_ready.v1.
type EnqueueProbeHandler struct {
	jobs port.JobPublisher
}

func NewEnqueueProbeHandler(jobs port.JobPublisher) *EnqueueProbeHandler {
	return &EnqueueProbeHandler{jobs: jobs}
}

func (h *EnqueueProbeHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaSourceReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.MediaID == "" || evt.SourceKey == "" {
		return messaging.NonRetryable(fmt.Errorf("media.source_ready payload missing mediaId or sourceKey"))
	}
	if err := h.jobs.PublishProbeJob(ctx, evt.MediaID, evt.SourceKey); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

// EnqueueAfterProbedHandler fans out audio ∥ proxy after media.probed.v1.
type EnqueueAfterProbedHandler struct {
	jobs port.JobPublisher
}

func NewEnqueueAfterProbedHandler(jobs port.JobPublisher) *EnqueueAfterProbedHandler {
	return &EnqueueAfterProbedHandler{jobs: jobs}
}

func (h *EnqueueAfterProbedHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaProbed
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.MediaID == "" || evt.SourceKey == "" {
		return messaging.NonRetryable(fmt.Errorf("media.probed payload missing mediaId or sourceKey"))
	}

	var firstErr error
	if evt.HasAudio {
		if err := h.jobs.PublishAudioJob(ctx, evt.MediaID, evt.SourceKey); err != nil {
			firstErr = err
		}
	}
	if !evt.SkipProxy {
		if err := h.jobs.PublishProxyJob(ctx, evt.MediaID, evt.SourceKey); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return messaging.Retryable(firstErr)
	}
	return nil
}

// EnqueueAfterAudioReadyHandler fans out waveform ∥ silence using media configuration.
type EnqueueAfterAudioReadyHandler struct {
	jobs  port.JobPublisher
	media domainmedia.WriteRepository
}

func NewEnqueueAfterAudioReadyHandler(jobs port.JobPublisher, media domainmedia.WriteRepository) *EnqueueAfterAudioReadyHandler {
	return &EnqueueAfterAudioReadyHandler{jobs: jobs, media: media}
}

func (h *EnqueueAfterAudioReadyHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaAudioReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.MediaID == "" || evt.AnalysisKey == "" {
		return messaging.NonRetryable(fmt.Errorf("media.audio_ready payload missing mediaId or analysisKey"))
	}

	var firstErr error
	if err := h.jobs.PublishWaveformJob(ctx, evt.MediaID, evt.AnalysisKey); err != nil {
		firstErr = err
	}

	mediaID, err := uuid.Parse(evt.MediaID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	cfg, err := h.media.GetConfiguration(ctx, mediaID)
	if err != nil {
		return messaging.Retryable(err)
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(mediaID)
	}
	if cfg.SilenceEnabled {
		silenceAttempt := 1
		if step, err := h.media.GetStep(ctx, mediaID, domainmedia.StepSilence); err == nil && step != nil && step.Attempt >= 1 {
			silenceAttempt = step.Attempt
		}
		if err := h.jobs.PublishSilenceJob(ctx, evt.MediaID, evt.AnalysisKey, port.SilenceJobOptions{
			ThresholdMode:      cfg.ThresholdMode,
			SilenceThresholdDb: cfg.SilenceThresholdDB,
			NoiseFloorOffsetDb: cfg.NoiseFloorOffsetDb(),
			Attempt:            silenceAttempt,
		}); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	txKey := evt.TranscriptionKey
	if txKey == "" {
		txKey = evt.AnalysisKey
	}
	if err := h.jobs.PublishTranscribeJob(ctx, evt.MediaID, txKey, port.TranscribeJobOptions{
		Language: cfg.Language,
	}); err != nil && firstErr == nil {
		firstErr = err
	}

	if firstErr != nil {
		return messaging.Retryable(firstErr)
	}
	return nil
}

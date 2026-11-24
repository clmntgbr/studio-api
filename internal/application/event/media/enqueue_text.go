package media

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mediacmd "go-api/internal/application/command/media"
	"go-api/internal/application/messaging"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

// EnqueueTextHandler starts text detection after media.transcribed.v1.
type EnqueueTextHandler struct {
	jobs  port.JobPublisher
	media domainmedia.WriteRepository
}

func NewEnqueueTextHandler(jobs port.JobPublisher, media domainmedia.WriteRepository) *EnqueueTextHandler {
	return &EnqueueTextHandler{jobs: jobs, media: media}
}

func (h *EnqueueTextHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaTranscribed
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.MediaID == "" || evt.WordsKey == "" {
		return messaging.NonRetryable(fmt.Errorf("media.transcribed payload missing mediaId or wordsKey"))
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
	if !cfg.FillersEnabled && !cfg.RepetitionsEnabled && !cfg.FalseStartsEnabled {
		return nil
	}
	lang := evt.Language
	if lang == "" && cfg.Language != nil {
		lang = *cfg.Language
	}
	if err := mediacmd.QueueStep(ctx, h.media, mediaID, domainmedia.StepText, time.Now().UTC()); err != nil {
		return messaging.Retryable(err)
	}
	if err := h.jobs.PublishTextJob(ctx, evt.MediaID, evt.WordsKey, port.TextJobOptions{
		Language:           lang,
		DetectionLevel:     cfg.DetectionLevel,
		FillersEnabled:     cfg.FillersEnabled,
		RepetitionsEnabled: cfg.RepetitionsEnabled,
		FalseStartsEnabled: cfg.FalseStartsEnabled,
	}); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

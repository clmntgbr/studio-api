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

// EnqueueLLMHandler starts LLM analysis after media.transcribed.v1.
type EnqueueLLMHandler struct {
	jobs  port.JobPublisher
	media domainmedia.WriteRepository
}

func NewEnqueueLLMHandler(jobs port.JobPublisher, media domainmedia.WriteRepository) *EnqueueLLMHandler {
	return &EnqueueLLMHandler{jobs: jobs, media: media}
}

func (h *EnqueueLLMHandler) Handle(ctx context.Context, payload []byte) error {
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

	lang := evt.Language
	if lang == "" && cfg.Language != nil {
		lang = *cfg.Language
	}
	if err := mediacmd.QueueStep(ctx, h.media, mediaID, domainmedia.StepLLM, time.Now().UTC()); err != nil {
		return messaging.Retryable(err)
	}
	if err := h.jobs.PublishLLMJob(ctx, evt.MediaID, evt.WordsKey, port.LLMJobOptions{
		Language:       lang,
		Tasks:          []string{"sponsors", "chapters", "viral"},
		SponsorsAction: string(cfg.SponsorsAction),
	}); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

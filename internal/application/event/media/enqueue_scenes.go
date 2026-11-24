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

// EnqueueScenesHandler starts ad/scene detection after media.proxy_ready.v1.
type EnqueueScenesHandler struct {
	jobs  port.JobPublisher
	media domainmedia.WriteRepository
}

func NewEnqueueScenesHandler(jobs port.JobPublisher, media domainmedia.WriteRepository) *EnqueueScenesHandler {
	return &EnqueueScenesHandler{jobs: jobs, media: media}
}

func (h *EnqueueScenesHandler) Handle(ctx context.Context, payload []byte) error {
	var evt domainmedia.MediaProxyReady
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	if evt.MediaID == "" || evt.ProxyKey == "" {
		return messaging.NonRetryable(fmt.Errorf("media.proxy_ready payload missing mediaId or proxyKey"))
	}
	mediaID, err := uuid.Parse(evt.MediaID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	if err := mediacmd.QueueStep(ctx, h.media, mediaID, domainmedia.StepScenes, time.Now().UTC()); err != nil {
		return messaging.Retryable(err)
	}
	attempt := 1
	if step, err := h.media.GetStep(ctx, mediaID, domainmedia.StepScenes); err == nil && step != nil && step.Attempt > 0 {
		attempt = step.Attempt
	}
	if err := h.jobs.PublishScenesJob(ctx, evt.MediaID, evt.ProxyKey, attempt); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

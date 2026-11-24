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

// EnqueueReframeHandler starts face tracking after media.proxy_ready.v1.
type EnqueueReframeHandler struct {
	jobs  port.JobPublisher
	media domainmedia.WriteRepository
}

func NewEnqueueReframeHandler(jobs port.JobPublisher, media domainmedia.WriteRepository) *EnqueueReframeHandler {
	return &EnqueueReframeHandler{jobs: jobs, media: media}
}

func (h *EnqueueReframeHandler) Handle(ctx context.Context, payload []byte) error {
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
	if err := mediacmd.QueueStep(ctx, h.media, mediaID, domainmedia.StepReframe, time.Now().UTC()); err != nil {
		return messaging.Retryable(err)
	}
	if err := h.jobs.PublishReframeJob(ctx, evt.MediaID, evt.ProxyKey); err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

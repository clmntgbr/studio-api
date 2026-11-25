package media

import (
	"context"
	"encoding/json"

	cmdtimeline "go-api/internal/application/command/timeline"
	"go-api/internal/application/messaging"
	"go-api/internal/application/realtime"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type RebuildTimelineHandler struct {
	rebuild   *cmdtimeline.RebuildHandler
	publisher *realtime.Publisher
}

func NewRebuildTimelineHandler(
	rebuild *cmdtimeline.RebuildHandler,
	transport port.RealtimePublisher,
) *RebuildTimelineHandler {
	return &RebuildTimelineHandler{
		rebuild:   rebuild,
		publisher: realtime.NewPublisher(transport),
	}
}

func (h *RebuildTimelineHandler) OnDetectionsUpdated(ctx context.Context, payload []byte) error {
	var evt domainmedia.DetectionsUpdated
	if err := json.Unmarshal(payload, &evt); err != nil {
		return messaging.NonRetryable(err)
	}
	mediaID, err := uuid.Parse(evt.MediaID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	projectID, err := uuid.Parse(evt.ProjectID)
	if err != nil {
		return messaging.NonRetryable(err)
	}
	tl, _, err := h.rebuild.Handle(ctx, cmdtimeline.RebuildCommand{
		ProjectID: projectID,
		MediaID:   mediaID,
	})
	if err != nil {
		return messaging.Retryable(err)
	}

	version := 0
	durationMs := int64(0)
	fingerprint := ""
	if tl != nil {
		version = tl.Version
		durationMs = tl.DurationMs
		fingerprint = tl.Fingerprint
	}

	// Always notify the editor so it can reload after config/detection rebuilds,
	// even when the fingerprint was unchanged.
	if err := h.publisher.ToProject(ctx, realtime.EntityTimeline, realtime.ActionUpdated, evt.ProjectID, map[string]any{
		"mediaId":     evt.MediaID,
		"projectId":   evt.ProjectID,
		"kind":        evt.Kind,
		"version":     version,
		"durationMs":  durationMs,
		"fingerprint": fingerprint,
	}); err != nil {
		return err
	}
	return nil
}

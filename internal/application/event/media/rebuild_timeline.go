package media

import (
	"context"
	"encoding/json"

	cmdtimeline "go-api/internal/application/command/timeline"
	"go-api/internal/application/messaging"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type RebuildTimelineHandler struct {
	rebuild *cmdtimeline.RebuildHandler
}

func NewRebuildTimelineHandler(rebuild *cmdtimeline.RebuildHandler) *RebuildTimelineHandler {
	return &RebuildTimelineHandler{rebuild: rebuild}
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
	_, _, err = h.rebuild.Handle(ctx, cmdtimeline.RebuildCommand{
		ProjectID: projectID,
		MediaID:   mediaID,
	})
	if err != nil {
		return messaging.Retryable(err)
	}
	return nil
}

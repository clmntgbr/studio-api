package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type MarkFetchFailedCommand struct {
	MediaID uuid.UUID
}

type MarkFetchFailedHandler struct {
	media domainmedia.WriteRepository
}

func NewMarkFetchFailedHandler(media domainmedia.WriteRepository) *MarkFetchFailedHandler {
	return &MarkFetchFailedHandler{media: media}
}

func (h *MarkFetchFailedHandler) Handle(ctx context.Context, cmd MarkFetchFailedCommand) error {
	if cmd.MediaID == uuid.Nil {
		return errors.New("mediaId is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}
	if m.Status == domainmedia.StatusFailed || m.Status == domainmedia.StatusReady || m.Status == domainmedia.StatusProcessing {
		return nil
	}
	m.MarkFailed()
	m.UpdatedAt = time.Now().UTC()
	if err := h.media.Update(ctx, m); err != nil {
		return errors.New("failed to mark fetch failed")
	}
	return nil
}

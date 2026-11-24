package detection

import (
	"context"
	"errors"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type DeleteOverrideCommand struct {
	OverrideID uuid.UUID
}

type DeleteOverrideHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	detectionR domaindetection.ReadRepository
	outbox     port.OutboxRepository
}

func NewDeleteOverrideHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	detectionR domaindetection.ReadRepository,
	outbox port.OutboxRepository,
) *DeleteOverrideHandler {
	return &DeleteOverrideHandler{
		media:      media,
		detections: detections,
		detectionR: detectionR,
		outbox:     outbox,
	}
}

func (h *DeleteOverrideHandler) Handle(ctx context.Context, cmd DeleteOverrideCommand) error {
	o, err := h.detectionR.GetOverrideByID(ctx, cmd.OverrideID)
	if err != nil {
		return err
	}
	if o == nil {
		return errors.New("override not found")
	}

	m, err := h.media.GetByID(ctx, o.MediaFileID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	eventKind := "override"
	if o.Kind != nil {
		eventKind = string(*o.Kind)
	}
	m.RecordDetectionsUpdated(eventKind)

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.detections.DeleteOverride(txCtx, o.ID); err != nil {
			return err
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return errors.New("failed to delete override")
	}
	return nil
}

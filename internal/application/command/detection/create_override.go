package detection

import (
	"context"
	"errors"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type CreateOverrideCommand struct {
	MediaID       uuid.UUID
	Action        string
	SourceStartMs int64
	SourceEndMs   int64
	Kind          *string
}

type CreateOverrideHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	outbox     port.OutboxRepository
}

func NewCreateOverrideHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	outbox port.OutboxRepository,
) *CreateOverrideHandler {
	return &CreateOverrideHandler{media: media, detections: detections, outbox: outbox}
}

func (h *CreateOverrideHandler) Handle(ctx context.Context, cmd CreateOverrideCommand) (*domaindetection.Override, error) {
	if cmd.SourceEndMs <= cmd.SourceStartMs {
		return nil, errors.New("sourceEndMs must be greater than sourceStartMs")
	}
	action := domaindetection.DecisionAction(cmd.Action)
	if action != domaindetection.ActionKeep && action != domaindetection.ActionRemove {
		return nil, errors.New("action must be keep or remove")
	}

	var kind *domaindetection.Kind
	if cmd.Kind != nil && *cmd.Kind != "" {
		k := domaindetection.Kind(*cmd.Kind)
		switch k {
		case domaindetection.KindSilence, domaindetection.KindFiller, domaindetection.KindRepetition,
			domaindetection.KindFalseStart, domaindetection.KindSponsor, domaindetection.KindAd:
			kind = &k
		default:
			return nil, errors.New("kind is invalid")
		}
	}

	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("media not found")
	}

	o := domaindetection.NewOverride(m.ID, action, cmd.SourceStartMs, cmd.SourceEndMs, kind)
	eventKind := "override"
	if kind != nil {
		eventKind = string(*kind)
	}
	m.RecordDetectionsUpdated(eventKind)

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.detections.SaveOverride(txCtx, o); err != nil {
			return err
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return nil, errors.New("failed to create override")
	}
	return o, nil
}

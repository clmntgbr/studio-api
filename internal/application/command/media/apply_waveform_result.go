package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ApplyWaveformResultCommand struct {
	MediaID  uuid.UUID
	PeaksKey string
}

type ApplyWaveformResultHandler struct {
	media  domainmedia.WriteRepository
	outbox port.OutboxRepository
}

func NewApplyWaveformResultHandler(media domainmedia.WriteRepository, outbox port.OutboxRepository) *ApplyWaveformResultHandler {
	return &ApplyWaveformResultHandler{media: media, outbox: outbox}
}

func (h *ApplyWaveformResultHandler) Handle(ctx context.Context, cmd ApplyWaveformResultCommand) error {
	if cmd.PeaksKey == "" {
		return errors.New("peaksKey is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepWaveform)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	m.RecordWaveformReady(cmd.PeaksKey)
	now := time.Now().UTC()

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := h.media.SaveAsset(txCtx, domainmedia.NewAsset(m.ID, domainmedia.AssetKindPeaks, cmd.PeaksKey, nil, nil)); err != nil {
			return err
		}
		if err := completeStep(txCtx, h.media, m.ID, domainmedia.StepWaveform, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return err
	}
	return tryMarkMediaReady(ctx, h.media, h.outbox, cmd.MediaID)
}

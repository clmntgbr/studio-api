package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ApplyProbeResultCommand struct {
	MediaID    uuid.UUID
	DurationMs int64
	Width      int
	Height     int
	FPS        float64
	VideoCodec string
	AudioCodec string
	HasAudio   bool
}

type ApplyProbeResultHandler struct {
	media  domainmedia.WriteRepository
	outbox port.OutboxRepository
}

func NewApplyProbeResultHandler(media domainmedia.WriteRepository, outbox port.OutboxRepository) *ApplyProbeResultHandler {
	return &ApplyProbeResultHandler{media: media, outbox: outbox}
}

func (h *ApplyProbeResultHandler) Handle(ctx context.Context, cmd ApplyProbeResultCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	// Idempotent: already probed (event already emitted once).
	if m.DurationMs != nil && *m.DurationMs > 0 && m.Width != nil {
		return nil
	}

	m.ApplyProbe(cmd.DurationMs, cmd.Width, cmd.Height, cmd.FPS, cmd.VideoCodec, cmd.AudioCodec, cmd.HasAudio)

	now := time.Now().UTC()
	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := completeStep(txCtx, h.media, m.ID, domainmedia.StepProbe, now); err != nil {
			return err
		}
		if cmd.HasAudio {
			if err := queueStep(txCtx, h.media, m.ID, domainmedia.StepAudio, now); err != nil {
				return err
			}
		}
		if err := queueStep(txCtx, h.media, m.ID, domainmedia.StepProxy, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return err
	}
	return tryMarkMediaReady(ctx, h.media, h.outbox, cmd.MediaID)
}

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

	now := time.Now().UTC()
	skipProxy := !domainmedia.NeedsProxyEncode(cmd.Width, cmd.Height)

	finishSkipProxy := func(txCtx context.Context) error {
		if m.StorageKey == nil || *m.StorageKey == "" {
			return errors.New("media source not ready")
		}
		key := *m.StorageKey
		if err := h.media.SaveAsset(txCtx, domainmedia.NewAsset(m.ID, domainmedia.AssetKindProxy, key, nil, nil)); err != nil {
			return err
		}
		return completeStep(txCtx, h.media, m.ID, domainmedia.StepProxy, now)
	}

	// Idempotent: already probed. Still heal skip-proxy if StepProxy never completed.
	if m.DurationMs != nil && *m.DurationMs > 0 && m.Width != nil {
		if !skipProxy {
			return nil
		}
		step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepProxy)
		if err != nil {
			return err
		}
		if step != nil && step.Status == domainmedia.StepStatusCompleted {
			return nil
		}
		return h.media.WithTransaction(ctx, func(txCtx context.Context) error {
			return finishSkipProxy(txCtx)
		})
	}

	m.ApplyProbe(cmd.DurationMs, cmd.Width, cmd.Height, cmd.FPS, cmd.VideoCodec, cmd.AudioCodec, cmd.HasAudio)

	if skipProxy {
		if m.StorageKey == nil || *m.StorageKey == "" {
			return errors.New("media source not ready")
		}
		// Source is already ≤720p-class: reuse it as the proxy asset (no encode job).
		m.RecordProxyReady(*m.StorageKey)
	}

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
		if skipProxy {
			if err := finishSkipProxy(txCtx); err != nil {
				return err
			}
		} else if err := queueStep(txCtx, h.media, m.ID, domainmedia.StepProxy, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return err
	}
	return tryMarkMediaReady(ctx, h.media, h.outbox, cmd.MediaID)
}

package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ApplyProxyResultCommand struct {
	MediaID  uuid.UUID
	ProxyKey string
}

type ApplyProxyResultHandler struct {
	media  domainmedia.WriteRepository
	outbox port.OutboxRepository
}

func NewApplyProxyResultHandler(media domainmedia.WriteRepository, outbox port.OutboxRepository) *ApplyProxyResultHandler {
	return &ApplyProxyResultHandler{media: media, outbox: outbox}
}

func (h *ApplyProxyResultHandler) Handle(ctx context.Context, cmd ApplyProxyResultCommand) error {
	if cmd.ProxyKey == "" {
		return errors.New("proxyKey is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepProxy)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	m.RecordProxyReady(cmd.ProxyKey)
	now := time.Now().UTC()

	return h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := h.media.SaveAsset(txCtx, domainmedia.NewAsset(m.ID, domainmedia.AssetKindProxy, cmd.ProxyKey, nil, nil)); err != nil {
			return err
		}
		if err := completeStep(txCtx, h.media, m.ID, domainmedia.StepProxy, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
}

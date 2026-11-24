package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

// MarkSourceReadyCommand is the single post-ingress entry point:
// upload (presign + MinIO webhook) and fetch (result.fetch.succeeded) both land here.
// Probe is enqueued by the outbox consumer on media.source_ready.v1 (durable, retried).
type MarkSourceReadyCommand struct {
	MediaID   uuid.UUID
	SourceKey string
	SizeBytes *int64
	Title     *string
}

type MarkSourceReadyHandler struct {
	media  domainmedia.WriteRepository
	outbox port.OutboxRepository
}

func NewMarkSourceReadyHandler(
	media domainmedia.WriteRepository,
	outbox port.OutboxRepository,
) *MarkSourceReadyHandler {
	return &MarkSourceReadyHandler{media: media, outbox: outbox}
}

func (h *MarkSourceReadyHandler) Handle(ctx context.Context, cmd MarkSourceReadyCommand) error {
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

	// Already past ingress: still accept late metadata (fetch title/size after webhook).
	if m.Status == domainmedia.StatusProcessing || m.Status == domainmedia.StatusReady {
		return h.applyLateMetadata(ctx, m, cmd)
	}
	if m.Status != domainmedia.StatusPending && m.Status != domainmedia.StatusFetching {
		return errors.New("media is not awaiting source")
	}

	key := cmd.SourceKey
	if key == "" && m.StorageKey != nil {
		key = *m.StorageKey
	}
	if key == "" {
		return errors.New("sourceKey is required")
	}
	if cmd.SizeBytes != nil {
		m.SizeBytes = cmd.SizeBytes
	}
	if cmd.Title != nil && *cmd.Title != "" {
		m.Title = cmd.Title
	}
	m.MarkSourceReady(key)

	now := time.Now().UTC()
	probeStep := domainmedia.NewProcessingStep(m.ID, domainmedia.StepProbe)
	probeStep.Status = domainmedia.StepStatusQueued
	probeStep.QueuedAt = &now

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		claimed, err := h.media.UpdateFromStatuses(txCtx, m, []domainmedia.Status{
			domainmedia.StatusPending,
			domainmedia.StatusFetching,
		})
		if err != nil {
			return err
		}
		if !claimed {
			return errAlreadyClaimed
		}
		if err := h.media.SaveStep(txCtx, probeStep); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if errors.Is(err, errAlreadyClaimed) {
		latest, getErr := h.media.GetByID(ctx, cmd.MediaID)
		if getErr != nil {
			return getErr
		}
		if latest == nil {
			return errors.New("media not found")
		}
		return h.applyLateMetadata(ctx, latest, cmd)
	}
	if err != nil {
		return errors.New("failed to mark source ready")
	}
	return nil
}

func (h *MarkSourceReadyHandler) applyLateMetadata(ctx context.Context, m *domainmedia.MediaFile, cmd MarkSourceReadyCommand) error {
	changed := false
	if cmd.SizeBytes != nil && (m.SizeBytes == nil || *m.SizeBytes != *cmd.SizeBytes) {
		m.SizeBytes = cmd.SizeBytes
		changed = true
	}
	if cmd.Title != nil && *cmd.Title != "" && (m.Title == nil || *m.Title != *cmd.Title) {
		m.Title = cmd.Title
		changed = true
	}
	if cmd.SourceKey != "" && (m.StorageKey == nil || *m.StorageKey != cmd.SourceKey) {
		key := cmd.SourceKey
		m.StorageKey = &key
		changed = true
	}
	if !changed {
		return nil
	}
	m.UpdatedAt = time.Now().UTC()
	if err := h.media.Update(ctx, m); err != nil {
		return errors.New("failed to update media metadata")
	}
	return nil
}

var errAlreadyClaimed = errors.New("media source already claimed")

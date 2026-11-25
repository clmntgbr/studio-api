package media

import (
	"context"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

// tryMarkMediaReady sets status=ready when the media is editable:
// probe always; audio + waveform when the file has audio.
// Proxy is intentionally NOT required — it can finish later for better playback / reframe.
func tryMarkMediaReady(ctx context.Context, repo domainmedia.WriteRepository, outbox port.OutboxRepository, mediaID uuid.UUID) error {
	m, err := repo.GetByID(ctx, mediaID)
	if err != nil || m == nil {
		return err
	}
	if m.Status == domainmedia.StatusReady || m.Status == domainmedia.StatusFailed {
		return nil
	}

	required := []domainmedia.StepKind{domainmedia.StepProbe}
	hasAudio := m.HasAudio != nil && *m.HasAudio
	if hasAudio {
		required = append(required, domainmedia.StepAudio, domainmedia.StepWaveform)
	}

	for _, kind := range required {
		step, err := repo.GetStep(ctx, mediaID, kind)
		if err != nil {
			return err
		}
		if step == nil || step.Status != domainmedia.StepStatusCompleted {
			return nil
		}
	}

	m.MarkReady()
	return repo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := repo.Update(txCtx, m); err != nil {
			return err
		}
		return outbox.StoreEvents(txCtx, m.PullEvents())
	})
}

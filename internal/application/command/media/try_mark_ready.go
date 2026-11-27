package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

// ErrMediaNotReady is returned when a mutation requires ready/rendering status.
var ErrMediaNotReady = errors.New("media is not ready")

// ErrMediaNotEditable is returned when timeline/config edits require status=ready.
var ErrMediaNotEditable = errors.New("media is not editable")

// stepDone reports whether a step is finished for the ready gate.
func stepDone(status domainmedia.StepStatus) bool {
	return status == domainmedia.StepStatusCompleted ||
		status == domainmedia.StepStatusSkipped ||
		status == domainmedia.StepStatusCancelled
}

// requiredStepsForReady lists pipeline steps that must finish before media becomes editable.
// Excludes render / reframe / scenes (on-demand).
func requiredStepsForReady(m *domainmedia.MediaFile) []domainmedia.StepKind {
	required := []domainmedia.StepKind{
		domainmedia.StepProbe,
		domainmedia.StepProxy,
	}
	hasAudio := m.HasAudio != nil && *m.HasAudio
	if hasAudio {
		required = append(required,
			domainmedia.StepAudio,
			domainmedia.StepWaveform,
			domainmedia.StepSilence,
			domainmedia.StepTranscribe,
			domainmedia.StepText,
			domainmedia.StepLLM,
		)
	}
	return required
}

// TryMarkMediaReady sets status=ready when all required (non-render) steps are done.
// Only transitions from processing → ready. Ignores uploading/fetching/rendering/failed/ready.
func TryMarkMediaReady(ctx context.Context, repo domainmedia.WriteRepository, outbox port.OutboxRepository, mediaID uuid.UUID) error {
	return tryMarkMediaReady(ctx, repo, outbox, mediaID)
}

func tryMarkMediaReady(ctx context.Context, repo domainmedia.WriteRepository, outbox port.OutboxRepository, mediaID uuid.UUID) error {
	m, err := repo.GetByID(ctx, mediaID)
	if err != nil || m == nil {
		return err
	}
	if m.Status != domainmedia.StatusProcessing {
		return nil
	}

	for _, kind := range requiredStepsForReady(m) {
		step, err := repo.GetStep(ctx, mediaID, kind)
		if err != nil {
			return err
		}
		if step == nil || !stepDone(step.Status) {
			return nil
		}
	}

	m.MarkReady()
	return repo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := repo.Update(txCtx, m); err != nil {
			return err
		}
		if outbox == nil {
			return nil
		}
		return outbox.StoreEvents(txCtx, m.PullEvents())
	})
}

// MarkMediaProcessing moves ready/rendering media back to processing (reanalyze).
func MarkMediaProcessing(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID) error {
	m, err := repo.GetByID(ctx, mediaID)
	if err != nil || m == nil {
		return err
	}
	if m.Status != domainmedia.StatusReady && m.Status != domainmedia.StatusRendering {
		return nil
	}
	m.MarkProcessing()
	return repo.Update(ctx, m)
}

// MarkMediaRendering sets media to rendering when an encode is enqueued.
func MarkMediaRendering(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID) error {
	m, err := repo.GetByID(ctx, mediaID)
	if err != nil || m == nil {
		return err
	}
	if m.Status == domainmedia.StatusRendering {
		return nil
	}
	if m.Status != domainmedia.StatusReady {
		return ErrMediaNotReady
	}
	m.MarkRendering()
	return repo.Update(ctx, m)
}

// ReturnMediaToReadyIfIdle sets ready when no in-flight renders remain.
func ReturnMediaToReadyIfIdle(
	ctx context.Context,
	mediaRepo domainmedia.WriteRepository,
	inFlight func(context.Context, uuid.UUID) (bool, error),
	mediaID uuid.UUID,
) error {
	if mediaID == uuid.Nil || mediaRepo == nil || inFlight == nil {
		return nil
	}
	busy, err := inFlight(ctx, mediaID)
	if err != nil || busy {
		return err
	}
	m, err := mediaRepo.GetByID(ctx, mediaID)
	if err != nil || m == nil {
		return err
	}
	if m.Status != domainmedia.StatusRendering {
		return nil
	}
	m.ReturnToReady()
	return mediaRepo.Update(ctx, m)
}

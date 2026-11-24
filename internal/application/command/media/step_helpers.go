package media

import (
	"context"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

func completeStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		step.Status = domainmedia.StepStatusCompleted
		step.Progress = 1
		completed := at
		step.CompletedAt = &completed
		return repo.SaveStep(ctx, step)
	}
	if step.Status == domainmedia.StepStatusCompleted {
		return nil
	}
	step.Status = domainmedia.StepStatusCompleted
	step.Progress = 1
	completed := at
	step.CompletedAt = &completed
	return repo.UpdateStep(ctx, step)
}

func queueStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		step.Status = domainmedia.StepStatusQueued
		q := at
		step.QueuedAt = &q
		return repo.SaveStep(ctx, step)
	}
	if step.Status == domainmedia.StepStatusCompleted ||
		step.Status == domainmedia.StepStatusQueued ||
		step.Status == domainmedia.StepStatusRunning {
		return nil
	}
	step.Status = domainmedia.StepStatusQueued
	q := at
	step.QueuedAt = &q
	return repo.UpdateStep(ctx, step)
}

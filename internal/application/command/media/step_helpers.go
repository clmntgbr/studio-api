package media

import (
	"context"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

// CompleteStep marks a processing step completed and appends a timing run.
func CompleteStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	return completeStep(ctx, repo, mediaID, kind, at)
}

func completeStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	created := false
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		created = true
	}
	if !created && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	started := stepStartAt(step, at)
	step.StartedAt = &started
	step.Status = domainmedia.StepStatusCompleted
	step.Progress = 1
	completed := at
	step.CompletedAt = &completed
	if step.Attempt < 1 {
		step.Attempt = 1
	}

	run := domainmedia.NewProcessingStepRun(mediaID, kind, step.Attempt, started, at, domainmedia.StepStatusCompleted, nil)

	if created {
		if err := repo.SaveStep(ctx, step); err != nil {
			return err
		}
	} else {
		if err := repo.UpdateStep(ctx, step); err != nil {
			return err
		}
	}
	return repo.SaveStepRun(ctx, run)
}

// MarkStepRunning sets a step to running (first progress tick). Idempotent for completed steps.
func MarkStepRunning(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		step.Status = domainmedia.StepStatusRunning
		step.QueuedAt = &at
		step.StartedAt = &at
		return repo.SaveStep(ctx, step)
	}
	if step.Status == domainmedia.StepStatusCompleted ||
		step.Status == domainmedia.StepStatusFailed ||
		step.Status == domainmedia.StepStatusCancelled ||
		step.Status == domainmedia.StepStatusRunning {
		return nil
	}
	step.Status = domainmedia.StepStatusRunning
	if step.StartedAt == nil {
		step.StartedAt = &at
	}
	return repo.UpdateStep(ctx, step)
}

// FailStep marks a processing step as failed and records a timing run.
func FailStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time, errMsg string) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	created := false
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		created = true
	}
	if !created && (step.Status == domainmedia.StepStatusCompleted || step.Status == domainmedia.StepStatusFailed) {
		return nil
	}
	started := stepStartAt(step, at)
	step.StartedAt = &started
	step.Status = domainmedia.StepStatusFailed
	msg := errMsg
	if msg == "" {
		msg = "step failed"
	}
	step.ErrorMessage = &msg
	completed := at
	step.CompletedAt = &completed
	if step.Attempt < 1 {
		step.Attempt = 1
	}
	run := domainmedia.NewProcessingStepRun(mediaID, kind, step.Attempt, started, at, domainmedia.StepStatusFailed, &msg)
	if created {
		if err := repo.SaveStep(ctx, step); err != nil {
			return err
		}
	} else if err := repo.UpdateStep(ctx, step); err != nil {
		return err
	}
	return repo.SaveStepRun(ctx, run)
}

// stepStartAt picks a started timestamp that is never after `at` (DB check: completed_at >= started_at).
// Do not use CreatedAt: NewProcessingStep sets it to time.Now(), which can be after a captured `at`.
func stepStartAt(step *domainmedia.ProcessingStep, at time.Time) time.Time {
	started := at
	if step.StartedAt != nil {
		started = *step.StartedAt
	} else if step.QueuedAt != nil {
		started = *step.QueuedAt
	}
	if started.After(at) {
		return at
	}
	return started
}

// QueueStep marks a processing step as queued.
func QueueStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	return queueStep(ctx, repo, mediaID, kind, at)
}

func queueStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		step.Status = domainmedia.StepStatusQueued
		step.Attempt = 1 // align with studiojobs NewJob(..., attempt=1)
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
	if step.Attempt < 1 {
		step.Attempt = 1
	}
	q := at
	step.QueuedAt = &q
	return repo.UpdateStep(ctx, step)
}

// EffectiveStepAttempt normalizes unset attempts to 1 (job publishers always send attempt≥1).
func EffectiveStepAttempt(attempt int) int {
	if attempt < 1 {
		return 1
	}
	return attempt
}

// RequeueStep resets a step to queued (render again / reanalyze). Always bumps attempt when the row exists.
func RequeueStep(ctx context.Context, repo domainmedia.WriteRepository, mediaID uuid.UUID, kind domainmedia.StepKind, at time.Time) error {
	step, err := repo.GetStep(ctx, mediaID, kind)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(mediaID, kind)
		step.Status = domainmedia.StepStatusQueued
		step.Attempt = 1
		q := at
		step.QueuedAt = &q
		return repo.SaveStep(ctx, step)
	}
	step.Status = domainmedia.StepStatusQueued
	step.Progress = 0
	step.ErrorMessage = nil
	step.StartedAt = nil
	step.CompletedAt = nil
	q := at
	step.QueuedAt = &q
	step.Attempt++
	if step.Attempt < 1 {
		step.Attempt = 1
	}
	return repo.UpdateStep(ctx, step)
}

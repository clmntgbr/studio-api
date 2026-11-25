package media

import (
	"testing"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

func TestStepStartAt_ignoresCreatedAtAfterCompletion(t *testing.T) {
	at := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	step := domainmedia.NewProcessingStep(uuid.New(), domainmedia.StepProxy)
	// NewProcessingStep stamps CreatedAt ~now, which would be after `at` in real skip-proxy flows.
	step.CreatedAt = at.Add(5 * time.Millisecond)

	got := stepStartAt(step, at)
	if !got.Equal(at) {
		t.Fatalf("stepStartAt = %v, want %v (must not use CreatedAt after at)", got, at)
	}
}

func TestStepStartAt_prefersStartedAt(t *testing.T) {
	at := time.Date(2026, 11, 25, 12, 0, 1, 0, time.UTC)
	started := at.Add(-2 * time.Second)
	step := &domainmedia.ProcessingStep{StartedAt: &started}
	got := stepStartAt(step, at)
	if !got.Equal(started) {
		t.Fatalf("stepStartAt = %v, want %v", got, started)
	}
}

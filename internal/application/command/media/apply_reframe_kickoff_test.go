package media

import (
	"context"
	"testing"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type recordingKickoff struct {
	calls []uuid.UUID
}

func (r *recordingKickoff) Handle(_ context.Context, mediaID uuid.UUID) error {
	r.calls = append(r.calls, mediaID)
	return nil
}

type fakeMediaReframe struct {
	m         *domainmedia.MediaFile
	step      *domainmedia.ProcessingStep
	savedCrop bool
}

func (f *fakeMediaReframe) WithTransaction(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}
func (f *fakeMediaReframe) Save(context.Context, *domainmedia.MediaFile) error     { panic("unused") }
func (f *fakeMediaReframe) Update(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeMediaReframe) UpdateFromStatuses(context.Context, *domainmedia.MediaFile, []domainmedia.Status) (bool, error) {
	panic("unused")
}
func (f *fakeMediaReframe) GetByID(context.Context, uuid.UUID) (*domainmedia.MediaFile, error) {
	return f.m, nil
}
func (f *fakeMediaReframe) Delete(context.Context, uuid.UUID) error { panic("unused") }
func (f *fakeMediaReframe) SaveAsset(context.Context, *domainmedia.Asset) error {
	f.savedCrop = true
	return nil
}
func (f *fakeMediaReframe) SaveConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaReframe) GetConfiguration(context.Context, uuid.UUID) (*domainmedia.Configuration, error) {
	panic("unused")
}
func (f *fakeMediaReframe) UpdateConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaReframe) GetAssetByKind(context.Context, uuid.UUID, string) (*domainmedia.Asset, error) {
	panic("unused")
}
func (f *fakeMediaReframe) SaveStep(_ context.Context, step *domainmedia.ProcessingStep) error {
	f.step = step
	return nil
}
func (f *fakeMediaReframe) UpdateStep(_ context.Context, step *domainmedia.ProcessingStep) error {
	f.step = step
	return nil
}
func (f *fakeMediaReframe) GetStep(context.Context, uuid.UUID, domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	return f.step, nil
}
func (f *fakeMediaReframe) SaveStepRun(context.Context, *domainmedia.ProcessingStepRun) error { return nil }

func TestApplyReframeResultHandler_callsKickoffAfterSuccess(t *testing.T) {
	ctx := context.Background()
	mediaID := uuid.New()
	kickoff := &recordingKickoff{}
	media := &fakeMediaReframe{m: &domainmedia.MediaFile{ID: mediaID}}
	h := NewApplyReframeResultHandler(media, kickoff)

	if err := h.Handle(ctx, ApplyReframeResultCommand{
		MediaID:     mediaID,
		CropPathKey: "crops/out.json",
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(kickoff.calls) != 1 || kickoff.calls[0] != mediaID {
		t.Fatalf("kickoff calls = %v, want [%s]", kickoff.calls, mediaID)
	}
}

func TestApplyReframeResultHandler_kickoffWhenStepAlreadyCompleted(t *testing.T) {
	ctx := context.Background()
	mediaID := uuid.New()
	kickoff := &recordingKickoff{}
	media := &fakeMediaReframe{
		m: &domainmedia.MediaFile{ID: mediaID},
		step: &domainmedia.ProcessingStep{
			MediaFileID: mediaID,
			Kind:        domainmedia.StepReframe,
			Status:      domainmedia.StepStatusCompleted,
		},
	}
	h := NewApplyReframeResultHandler(media, kickoff)

	if err := h.Handle(ctx, ApplyReframeResultCommand{
		MediaID:     mediaID,
		CropPathKey: "crops/out.json",
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if media.savedCrop {
		t.Fatal("expected no crop re-save when step already completed")
	}
	if len(kickoff.calls) != 1 || kickoff.calls[0] != mediaID {
		t.Fatalf("kickoff calls = %v, want [%s]", kickoff.calls, mediaID)
	}
}

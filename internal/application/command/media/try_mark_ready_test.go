package media

import (
	"context"
	"testing"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type fakeReadyMediaRepo struct {
	media  *domainmedia.MediaFile
	steps  map[domainmedia.StepKind]*domainmedia.ProcessingStep
	updated *domainmedia.MediaFile
}

func (f *fakeReadyMediaRepo) WithTransaction(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}
func (f *fakeReadyMediaRepo) Save(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeReadyMediaRepo) Update(_ context.Context, m *domainmedia.MediaFile) error {
	cp := *m
	f.updated = &cp
	f.media = &cp
	return nil
}
func (f *fakeReadyMediaRepo) UpdateFromStatuses(context.Context, *domainmedia.MediaFile, []domainmedia.Status) (bool, error) {
	panic("unused")
}
func (f *fakeReadyMediaRepo) GetByID(context.Context, uuid.UUID) (*domainmedia.MediaFile, error) {
	return f.media, nil
}
func (f *fakeReadyMediaRepo) Delete(context.Context, uuid.UUID) error { panic("unused") }
func (f *fakeReadyMediaRepo) SaveAsset(context.Context, *domainmedia.Asset) error {
	panic("unused")
}
func (f *fakeReadyMediaRepo) SaveConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeReadyMediaRepo) GetConfiguration(context.Context, uuid.UUID) (*domainmedia.Configuration, error) {
	panic("unused")
}
func (f *fakeReadyMediaRepo) UpdateConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeReadyMediaRepo) GetAssetByKind(context.Context, uuid.UUID, string) (*domainmedia.Asset, error) {
	panic("unused")
}
func (f *fakeReadyMediaRepo) SaveStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeReadyMediaRepo) UpdateStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeReadyMediaRepo) GetStep(_ context.Context, _ uuid.UUID, kind domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	return f.steps[kind], nil
}
func (f *fakeReadyMediaRepo) SaveStepRun(context.Context, *domainmedia.ProcessingStepRun) error {
	panic("unused")
}

func TestRequiredStepsForReady_WithAudio(t *testing.T) {
	has := true
	m := &domainmedia.MediaFile{HasAudio: &has}
	got := requiredStepsForReady(m)
	want := []domainmedia.StepKind{
		domainmedia.StepProbe,
		domainmedia.StepProxy,
		domainmedia.StepAudio,
		domainmedia.StepWaveform,
		domainmedia.StepSilence,
		domainmedia.StepTranscribe,
		domainmedia.StepText,
		domainmedia.StepLLM,
	}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%s want %s", i, got[i], want[i])
		}
	}
}

func TestRequiredStepsForReady_NoAudio(t *testing.T) {
	has := false
	m := &domainmedia.MediaFile{HasAudio: &has}
	got := requiredStepsForReady(m)
	if len(got) != 2 || got[0] != domainmedia.StepProbe || got[1] != domainmedia.StepProxy {
		t.Fatalf("got=%v", got)
	}
}

func TestTryMarkMediaReady_WaitsForLLM(t *testing.T) {
	has := true
	id := uuid.New()
	repo := &fakeReadyMediaRepo{
		media: &domainmedia.MediaFile{ID: id, Status: domainmedia.StatusProcessing, HasAudio: &has},
		steps: map[domainmedia.StepKind]*domainmedia.ProcessingStep{},
	}
	for _, k := range requiredStepsForReady(repo.media) {
		if k == domainmedia.StepLLM {
			continue
		}
		repo.steps[k] = &domainmedia.ProcessingStep{Kind: k, Status: domainmedia.StepStatusCompleted}
	}
	if err := tryMarkMediaReady(context.Background(), repo, nil, id); err != nil {
		t.Fatal(err)
	}
	if repo.media.Status != domainmedia.StatusProcessing {
		t.Fatalf("status=%s want processing", repo.media.Status)
	}

	repo.steps[domainmedia.StepLLM] = &domainmedia.ProcessingStep{
		Kind: domainmedia.StepLLM, Status: domainmedia.StepStatusCompleted,
	}
	if err := tryMarkMediaReady(context.Background(), repo, nil, id); err != nil {
		t.Fatal(err)
	}
	if repo.media.Status != domainmedia.StatusReady {
		t.Fatalf("status=%s want ready", repo.media.Status)
	}
}

func TestTryMarkMediaReady_IgnoresRendering(t *testing.T) {
	id := uuid.New()
	has := false
	repo := &fakeReadyMediaRepo{
		media: &domainmedia.MediaFile{ID: id, Status: domainmedia.StatusRendering, HasAudio: &has},
		steps: map[domainmedia.StepKind]*domainmedia.ProcessingStep{
			domainmedia.StepProbe: {Status: domainmedia.StepStatusCompleted},
			domainmedia.StepProxy: {Status: domainmedia.StepStatusCompleted},
		},
	}
	if err := tryMarkMediaReady(context.Background(), repo, nil, id); err != nil {
		t.Fatal(err)
	}
	if repo.media.Status != domainmedia.StatusRendering {
		t.Fatalf("status=%s", repo.media.Status)
	}
}

func TestMarkMediaRendering_FromReady(t *testing.T) {
	id := uuid.New()
	repo := &fakeReadyMediaRepo{
		media: &domainmedia.MediaFile{ID: id, Status: domainmedia.StatusReady},
	}
	if err := MarkMediaRendering(context.Background(), repo, id); err != nil {
		t.Fatal(err)
	}
	if repo.media.Status != domainmedia.StatusRendering {
		t.Fatalf("status=%s", repo.media.Status)
	}
}

func TestMarkMediaRendering_FromProcessing(t *testing.T) {
	id := uuid.New()
	repo := &fakeReadyMediaRepo{
		media: &domainmedia.MediaFile{ID: id, Status: domainmedia.StatusProcessing},
	}
	if err := MarkMediaRendering(context.Background(), repo, id); err != ErrMediaNotReady {
		t.Fatalf("err=%v", err)
	}
}

func TestReturnMediaToReadyIfIdle(t *testing.T) {
	id := uuid.New()
	repo := &fakeReadyMediaRepo{
		media: &domainmedia.MediaFile{ID: id, Status: domainmedia.StatusRendering},
	}
	if err := ReturnMediaToReadyIfIdle(context.Background(), repo, func(context.Context, uuid.UUID) (bool, error) {
		return false, nil
	}, id); err != nil {
		t.Fatal(err)
	}
	if repo.media.Status != domainmedia.StatusReady {
		t.Fatalf("status=%s", repo.media.Status)
	}
}

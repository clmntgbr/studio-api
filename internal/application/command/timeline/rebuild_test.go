package timeline

import (
	"context"
	"testing"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	domaintimeline "go-api/internal/domain/timeline"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type fakeMediaWrite struct {
	file *domainmedia.MediaFile
}

func (f *fakeMediaWrite) WithTransaction(context.Context, func(context.Context) error) error {
	panic("unused")
}
func (f *fakeMediaWrite) Save(context.Context, *domainmedia.MediaFile) error   { panic("unused") }
func (f *fakeMediaWrite) Update(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeMediaWrite) UpdateFromStatuses(context.Context, *domainmedia.MediaFile, []domainmedia.Status) (bool, error) {
	panic("unused")
}
func (f *fakeMediaWrite) GetByID(_ context.Context, id uuid.UUID) (*domainmedia.MediaFile, error) {
	if f.file == nil || f.file.ID != id {
		return nil, nil
	}
	return f.file, nil
}
func (f *fakeMediaWrite) Delete(context.Context, uuid.UUID) error             { panic("unused") }
func (f *fakeMediaWrite) SaveAsset(context.Context, *domainmedia.Asset) error { panic("unused") }
func (f *fakeMediaWrite) SaveConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaWrite) GetConfiguration(context.Context, uuid.UUID) (*domainmedia.Configuration, error) {
	return nil, nil
}
func (f *fakeMediaWrite) UpdateConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaWrite) GetAssetByKind(context.Context, uuid.UUID, string) (*domainmedia.Asset, error) {
	panic("unused")
}
func (f *fakeMediaWrite) SaveStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeMediaWrite) UpdateStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeMediaWrite) GetStep(context.Context, uuid.UUID, domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	panic("unused")
}
func (f *fakeMediaWrite) SaveStepRun(context.Context, *domainmedia.ProcessingStepRun) error {
	panic("unused")
}

type fakeDetectionRead struct {
	detections []domaindetection.Detection
}

func (f *fakeDetectionRead) ListByMedia(context.Context, uuid.UUID) ([]domaindetection.Detection, error) {
	return f.detections, nil
}
func (f *fakeDetectionRead) GetOverrideByID(context.Context, uuid.UUID) (*domaindetection.Override, error) {
	panic("unused")
}
func (f *fakeDetectionRead) ListOverridesByMedia(context.Context, uuid.UUID) ([]domaindetection.Override, error) {
	return nil, nil
}

type fakeTimelineWrite struct {
	saveCalls      int
	replaceCalls   int
	nextVersion    int
	supersedeCalls int
	lastSaved      *domaintimeline.Timeline
	lastReplaced   *domaintimeline.Timeline
}

func (f *fakeTimelineWrite) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (f *fakeTimelineWrite) Save(_ context.Context, t *domaintimeline.Timeline, _ []domaintimeline.Segment, _ []domaintimeline.Decision) error {
	f.saveCalls++
	f.lastSaved = t
	return nil
}
func (f *fakeTimelineWrite) ReplaceContents(_ context.Context, t *domaintimeline.Timeline, _ []domaintimeline.Segment, _ []domaintimeline.Decision) error {
	f.replaceCalls++
	f.lastReplaced = t
	return nil
}
func (f *fakeTimelineWrite) Update(context.Context, *domaintimeline.Timeline) error { panic("unused") }
func (f *fakeTimelineWrite) GetByID(context.Context, uuid.UUID) (*domaintimeline.Timeline, error) {
	panic("unused")
}
func (f *fakeTimelineWrite) SupersedeDrafts(context.Context, uuid.UUID) error {
	f.supersedeCalls++
	return nil
}
func (f *fakeTimelineWrite) NextVersion(context.Context, uuid.UUID) (int, error) {
	f.nextVersion++
	return 2, nil
}

type fakeTimelineRead struct {
	head *domaintimeline.Timeline
}

func (f *fakeTimelineRead) FindByID(context.Context, uuid.UUID) (*domaintimeline.Timeline, error) {
	panic("unused")
}
func (f *fakeTimelineRead) FindHeadByProject(context.Context, uuid.UUID) (*domaintimeline.Timeline, error) {
	return f.head, nil
}
func (f *fakeTimelineRead) ListByProject(context.Context, uuid.UUID) ([]domaintimeline.Timeline, error) {
	panic("unused")
}
func (f *fakeTimelineRead) ListSegments(context.Context, uuid.UUID) ([]domaintimeline.Segment, error) {
	panic("unused")
}
func (f *fakeTimelineRead) ListDecisions(context.Context, uuid.UUID) ([]domaintimeline.Decision, error) {
	panic("unused")
}

func testMedia(projectID, mediaID uuid.UUID) *domainmedia.MediaFile {
	dur := int64(10_000)
	return &domainmedia.MediaFile{
		ID:         mediaID,
		ProjectID:  projectID,
		DurationMs: &dur,
	}
}

func expectedFingerprint(mediaID uuid.UUID, dets []domaindetection.Detection) string {
	cfg := domainmedia.DefaultConfiguration(mediaID)
	in := eng.Input{
		MediaID:    mediaID,
		DurationMs: 10_000,
		Config: eng.Config{
			SilenceEnabled:       cfg.SilenceEnabled,
			SilenceMinDurationMs: cfg.SilenceMinDurationMs,
			PaddingBeforeMs:      cfg.PaddingBeforeMs,
			PaddingAfterMs:       cfg.PaddingAfterMs,
			SpeechMinDurationMs:  cfg.SpeechMinDurationMs,
			SponsorsAction:       eng.ActionRemove,
			AdsAction:            eng.ActionRemove,
		},
	}
	for _, d := range dets {
		in.Detections = append(in.Detections, eng.Detection{
			ID: d.ID, Kind: eng.DetectionKind(d.Kind), SourceStartMs: d.SourceStartMs, SourceEndMs: d.SourceEndMs,
		})
	}
	return eng.Build(in).Fingerprint
}

func TestRebuildHandler_Handle(t *testing.T) {
	projectID := uuid.New()
	mediaID := uuid.New()
	cmd := RebuildCommand{ProjectID: projectID, MediaID: mediaID}

	t.Run("first rebuild nil head saves version 1", func(t *testing.T) {
		tw := &fakeTimelineWrite{}
		h := NewRebuildHandler(
			&fakeMediaWrite{file: testMedia(projectID, mediaID)},
			&fakeDetectionRead{},
			tw,
			&fakeTimelineRead{},
		)
		tl, _, err := h.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		if tw.saveCalls != 1 || tw.replaceCalls != 0 {
			t.Fatalf("save=%d replace=%d", tw.saveCalls, tw.replaceCalls)
		}
		if tw.nextVersion != 0 || tw.supersedeCalls != 0 {
			t.Fatalf("nextVersion=%d supersede=%d", tw.nextVersion, tw.supersedeCalls)
		}
		if tl.Version != 1 {
			t.Fatalf("version=%d", tl.Version)
		}
	})

	t.Run("rebuild different fingerprint replaces same timeline", func(t *testing.T) {
		headID := uuid.New()
		detID := uuid.New()
		dets := []domaindetection.Detection{{
			ID: detID, Kind: domaindetection.KindSilence, SourceStartMs: 2000, SourceEndMs: 5000,
		}}
		fp := expectedFingerprint(mediaID, dets)
		tw := &fakeTimelineWrite{}
		head := &domaintimeline.Timeline{
			ID: headID, ProjectID: projectID, Version: 1, Status: domaintimeline.StatusValidated,
			Fingerprint: "stale-" + fp, DurationMs: 10_000,
		}
		h := NewRebuildHandler(
			&fakeMediaWrite{file: testMedia(projectID, mediaID)},
			&fakeDetectionRead{detections: dets},
			tw,
			&fakeTimelineRead{head: head},
		)
		tl, out, err := h.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		if tw.replaceCalls != 1 || tw.saveCalls != 0 {
			t.Fatalf("save=%d replace=%d", tw.saveCalls, tw.replaceCalls)
		}
		if tw.nextVersion != 0 || tw.supersedeCalls != 0 {
			t.Fatalf("nextVersion=%d supersede=%d", tw.nextVersion, tw.supersedeCalls)
		}
		if tl.ID != headID {
			t.Fatalf("timeline id changed: %s", tl.ID)
		}
		if tw.lastReplaced == nil || tw.lastReplaced.ID != headID {
			t.Fatal("ReplaceContents wrong timeline")
		}
		if tl.Status != domaintimeline.StatusDraft || tl.Fingerprint != out.Fingerprint {
			t.Fatalf("tl status=%s fp=%s", tl.Status, tl.Fingerprint)
		}
	})

	t.Run("same fingerprint skips persist", func(t *testing.T) {
		fp := expectedFingerprint(mediaID, nil)
		tw := &fakeTimelineWrite{}
		head := &domaintimeline.Timeline{
			ID: uuid.New(), ProjectID: projectID, Version: 1,
			Fingerprint: fp, DurationMs: 10_000,
		}
		h := NewRebuildHandler(
			&fakeMediaWrite{file: testMedia(projectID, mediaID)},
			&fakeDetectionRead{},
			tw,
			&fakeTimelineRead{head: head},
		)
		tl, _, err := h.Handle(context.Background(), cmd)
		if err != nil {
			t.Fatal(err)
		}
		if tw.saveCalls != 0 || tw.replaceCalls != 0 {
			t.Fatalf("save=%d replace=%d", tw.saveCalls, tw.replaceCalls)
		}
		if tl != head {
			t.Fatal("expected same head pointer")
		}
	})
}

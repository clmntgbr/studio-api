package render

import (
	"context"
	"encoding/json"
	"testing"

	domainmedia "go-api/internal/domain/media"
	domainpreset "go-api/internal/domain/outputpreset"
	domainrender "go-api/internal/domain/render"
	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
)

type fakeRenderReadRepo struct {
	byProject []domainrender.Render
}

func (f *fakeRenderReadRepo) FindByID(context.Context, uuid.UUID) (*domainrender.Render, error) {
	panic("unused")
}
func (f *fakeRenderReadRepo) FindByFingerprint(context.Context, uuid.UUID, string) (*domainrender.Render, error) {
	panic("unused")
}
func (f *fakeRenderReadRepo) ListByProject(_ context.Context, _ uuid.UUID) ([]domainrender.Render, error) {
	return f.byProject, nil
}

type fakeRenderWriteKickoff struct {
	updated []*domainrender.Render
}

func (f *fakeRenderWriteKickoff) WithTransaction(context.Context, func(context.Context) error) error {
	panic("unused")
}
func (f *fakeRenderWriteKickoff) Save(context.Context, *domainrender.Render) error { panic("unused") }
func (f *fakeRenderWriteKickoff) Update(_ context.Context, r *domainrender.Render) error {
	cp := *r
	f.updated = append(f.updated, &cp)
	return nil
}
func (f *fakeRenderWriteKickoff) GetByID(context.Context, uuid.UUID) (*domainrender.Render, error) {
	panic("unused")
}
func (f *fakeRenderWriteKickoff) FindByFingerprint(context.Context, uuid.UUID, string) (*domainrender.Render, error) {
	panic("unused")
}

type fakeMediaKickoff struct {
	mediaID   uuid.UUID
	projectID uuid.UUID
	source    string
	hasCrop   bool
}

func (f *fakeMediaKickoff) WithTransaction(context.Context, func(context.Context) error) error {
	panic("unused")
}
func (f *fakeMediaKickoff) Save(context.Context, *domainmedia.MediaFile) error   { panic("unused") }
func (f *fakeMediaKickoff) Update(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeMediaKickoff) UpdateFromStatuses(context.Context, *domainmedia.MediaFile, []domainmedia.Status) (bool, error) {
	panic("unused")
}
func (f *fakeMediaKickoff) GetByID(_ context.Context, id uuid.UUID) (*domainmedia.MediaFile, error) {
	if id != f.mediaID {
		return nil, nil
	}
	key := f.source
	return &domainmedia.MediaFile{ID: id, ProjectID: f.projectID, StorageKey: &key}, nil
}
func (f *fakeMediaKickoff) Delete(context.Context, uuid.UUID) error             { panic("unused") }
func (f *fakeMediaKickoff) SaveAsset(context.Context, *domainmedia.Asset) error { panic("unused") }
func (f *fakeMediaKickoff) SaveConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaKickoff) GetConfiguration(context.Context, uuid.UUID) (*domainmedia.Configuration, error) {
	panic("unused")
}
func (f *fakeMediaKickoff) UpdateConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaKickoff) GetAssetByKind(_ context.Context, mediaID uuid.UUID, kind string) (*domainmedia.Asset, error) {
	if mediaID != f.mediaID {
		return nil, nil
	}
	if kind == domainmedia.AssetKindCropPath && f.hasCrop {
		return &domainmedia.Asset{StorageKey: "crops/path.json"}, nil
	}
	return nil, nil
}
func (f *fakeMediaKickoff) SaveStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeMediaKickoff) UpdateStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeMediaKickoff) GetStep(context.Context, uuid.UUID, domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	panic("unused")
}
func (f *fakeMediaKickoff) SaveStepRun(context.Context, *domainmedia.ProcessingStepRun) error {
	panic("unused")
}

func markerPlan(mediaID uuid.UUID) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"awaitingReframe": true,
		"mediaId":         mediaID.String(),
	})
	return b
}

func TestKickoffDeferredHandler_queuesPendingRenderForMedia(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	timelineID := uuid.New()
	presetID := uuid.New()
	mediaID := uuid.New()
	renderID := uuid.New()

	marker := markerPlan(mediaID)
	rendersRead := &fakeRenderReadRepo{byProject: []domainrender.Render{{
		ID:             renderID,
		ProjectID:      projectID,
		TimelineID:     timelineID,
		OutputPresetID: presetID,
		Kind:           domainrender.KindMaster,
		Plan:           marker,
		Status:         domainrender.StatusPending,
	}}}
	rendersWrite := &fakeRenderWriteKickoff{}
	timelines := &fakeTimelineRead{
		head:     &domaintimeline.Timeline{ID: timelineID, ProjectID: projectID},
		segments: []domaintimeline.Segment{{MediaFileID: mediaID, SourceStartMs: 0, SourceEndMs: 5000}},
	}
	presets := &fakePresetRepo{
		preset: &domainpreset.Preset{ID: presetID, Width: 1080, Height: 1920, CropMode: "face_track", VideoCodec: "h264", AudioCodec: "aac"},
	}
	media := &fakeMediaKickoff{mediaID: mediaID, projectID: projectID, source: "sources/vid.mp4", hasCrop: true}
	jobs := &fakeJobs{}

	h := NewKickoffDeferredHandler(rendersWrite, rendersRead, media, QueuedPlanDeps(timelines, presets, media, nil, nil, nil, ""), jobs)
	if err := h.Handle(ctx, mediaID); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if jobs.render != 1 {
		t.Fatalf("PublishRenderJob calls = %d, want 1", jobs.render)
	}
	if len(rendersWrite.updated) != 1 {
		t.Fatalf("updates = %d, want 1", len(rendersWrite.updated))
	}
	up := rendersWrite.updated[0]
	if up.Status != domainrender.StatusQueued {
		t.Fatalf("status = %q, want queued", up.Status)
	}
	if !planHasOutput(up.Plan) {
		t.Fatalf("plan missing output segments: %s", string(up.Plan))
	}
}

func TestKickoffDeferredHandler_ignoresOtherMediaAndNonPending(t *testing.T) {
	ctx := context.Background()
	mediaA := uuid.New()
	mediaB := uuid.New()
	projectID := uuid.New()
	timelineID := uuid.New()
	presetID := uuid.New()

	rendersRead := &fakeRenderReadRepo{byProject: []domainrender.Render{
		{
			ID:             uuid.New(),
			ProjectID:      projectID,
			TimelineID:     timelineID,
			OutputPresetID: presetID,
			Kind:           domainrender.KindMaster,
			Plan:           markerPlan(mediaB),
			Status:         domainrender.StatusPending,
		},
		{
			ID:             uuid.New(),
			ProjectID:      projectID,
			TimelineID:     timelineID,
			OutputPresetID: presetID,
			Kind:           domainrender.KindMaster,
			Plan:           markerPlan(mediaA),
			Status:         domainrender.StatusQueued,
		},
	}}
	media := &fakeMediaKickoff{mediaID: mediaA, projectID: projectID, source: "sources/vid.mp4", hasCrop: true}
	jobs := &fakeJobs{}

	h := NewKickoffDeferredHandler(
		&fakeRenderWriteKickoff{},
		rendersRead,
		media,
		QueuedPlanDeps(
			&fakeTimelineRead{segments: []domaintimeline.Segment{{MediaFileID: mediaA}}},
			&fakePresetRepo{preset: &domainpreset.Preset{ID: presetID, Width: 1080, Height: 1920, CropMode: "none", VideoCodec: "h264", AudioCodec: "aac"}},
			media, nil, nil, nil, "",
		),
		jobs,
	)
	if err := h.Handle(ctx, mediaA); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if jobs.render != 0 {
		t.Fatalf("PublishRenderJob calls = %d, want 0", jobs.render)
	}
}

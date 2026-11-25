package render

import (
	"context"
	"encoding/json"
	"testing"

	domainmedia "go-api/internal/domain/media"
	domainpreset "go-api/internal/domain/outputpreset"
	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"
	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
)

type fakeRenderRepo struct {
	saved *domainrender.Render
}

func (f *fakeRenderRepo) WithTransaction(context.Context, func(context.Context) error) error {
	panic("unused")
}
func (f *fakeRenderRepo) Save(_ context.Context, r *domainrender.Render) error {
	f.saved = r
	return nil
}
func (f *fakeRenderRepo) Update(context.Context, *domainrender.Render) error { panic("unused") }
func (f *fakeRenderRepo) GetByID(context.Context, uuid.UUID) (*domainrender.Render, error) {
	panic("unused")
}
func (f *fakeRenderRepo) FindByFingerprint(context.Context, uuid.UUID, string) (*domainrender.Render, error) {
	return nil, nil
}

type fakeTimelineRead struct {
	head     *domaintimeline.Timeline
	segments []domaintimeline.Segment
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
	return f.segments, nil
}
func (f *fakeTimelineRead) ListDecisions(context.Context, uuid.UUID) ([]domaintimeline.Decision, error) {
	return nil, nil
}

type fakePresetRepo struct {
	preset *domainpreset.Preset
}

func (f *fakePresetRepo) FindByID(context.Context, uuid.UUID) (*domainpreset.Preset, error) {
	return f.preset, nil
}
func (f *fakePresetRepo) List(context.Context) ([]domainpreset.Preset, error) {
	panic("unused")
}

type fakeMediaWrite struct {
	mediaID  uuid.UUID
	source   string
	proxyKey string
	hasCrop  bool
}

func (f *fakeMediaWrite) WithTransaction(context.Context, func(context.Context) error) error {
	panic("unused")
}
func (f *fakeMediaWrite) Save(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeMediaWrite) Update(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeMediaWrite) UpdateFromStatuses(context.Context, *domainmedia.MediaFile, []domainmedia.Status) (bool, error) {
	panic("unused")
}
func (f *fakeMediaWrite) GetByID(_ context.Context, id uuid.UUID) (*domainmedia.MediaFile, error) {
	if id != f.mediaID {
		return nil, nil
	}
	key := f.source
	return &domainmedia.MediaFile{ID: id, StorageKey: &key}, nil
}
func (f *fakeMediaWrite) Delete(context.Context, uuid.UUID) error { panic("unused") }
func (f *fakeMediaWrite) SaveAsset(context.Context, *domainmedia.Asset) error { panic("unused") }
func (f *fakeMediaWrite) SaveConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaWrite) GetConfiguration(context.Context, uuid.UUID) (*domainmedia.Configuration, error) {
	panic("unused")
}
func (f *fakeMediaWrite) UpdateConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaWrite) GetAssetByKind(_ context.Context, mediaID uuid.UUID, kind string) (*domainmedia.Asset, error) {
	if mediaID != f.mediaID {
		return nil, nil
	}
	switch kind {
	case domainmedia.AssetKindProxy:
		if f.proxyKey == "" {
			return nil, nil
		}
		return &domainmedia.Asset{StorageKey: f.proxyKey}, nil
	case domainmedia.AssetKindCropPath:
		if !f.hasCrop {
			return nil, nil
		}
		return &domainmedia.Asset{StorageKey: "crops/path.json"}, nil
	default:
		return nil, nil
	}
}
func (f *fakeMediaWrite) SaveStep(_ context.Context, step *domainmedia.ProcessingStep) error {
	return nil
}
func (f *fakeMediaWrite) UpdateStep(context.Context, *domainmedia.ProcessingStep) error { panic("unused") }
func (f *fakeMediaWrite) GetStep(context.Context, uuid.UUID, domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	return nil, nil
}
func (f *fakeMediaWrite) SaveStepRun(context.Context, *domainmedia.ProcessingStepRun) error {
	panic("unused")
}

type reframeCall struct {
	mediaID string
	proxy   string
	w, h    int
}

type fakeJobs struct {
	reframe []reframeCall
	render  int
}

func (f *fakeJobs) PublishFetchJob(context.Context, string, string) error          { panic("unused") }
func (f *fakeJobs) PublishProbeJob(context.Context, string, string) error          { panic("unused") }
func (f *fakeJobs) PublishAudioJob(context.Context, string, string) error          { panic("unused") }
func (f *fakeJobs) PublishProxyJob(context.Context, string, string) error            { panic("unused") }
func (f *fakeJobs) PublishWaveformJob(context.Context, string, string) error         { panic("unused") }
func (f *fakeJobs) PublishSilenceJob(context.Context, string, string, port.SilenceJobOptions) error {
	panic("unused")
}
func (f *fakeJobs) PublishTranscribeJob(context.Context, string, string, port.TranscribeJobOptions) error {
	panic("unused")
}
func (f *fakeJobs) PublishTextJob(context.Context, string, string, port.TextJobOptions) error {
	panic("unused")
}
func (f *fakeJobs) PublishLLMJob(context.Context, string, string, port.LLMJobOptions) error {
	panic("unused")
}
func (f *fakeJobs) PublishScenesJob(context.Context, string, string, int) error      { panic("unused") }
func (f *fakeJobs) PublishRenderJob(context.Context, string, json.RawMessage) error {
	f.render++
	return nil
}
func (f *fakeJobs) PublishReframeJob(_ context.Context, mediaID, proxyKey string, w, h int) error {
	f.reframe = append(f.reframe, reframeCall{mediaID, proxyKey, w, h})
	return nil
}

func TestCreateHandler_defersWhenCropMissing_faceTrack(t *testing.T) {
	ctx := context.Background()
	projectID := uuid.New()
	timelineID := uuid.New()
	presetID := uuid.New()
	mediaID := uuid.New()

	renders := &fakeRenderRepo{}
	timelines := &fakeTimelineRead{
		head:     &domaintimeline.Timeline{ID: timelineID, ProjectID: projectID},
		segments: []domaintimeline.Segment{{MediaFileID: mediaID, SourceStartMs: 0, SourceEndMs: 5000}},
	}
	presets := &fakePresetRepo{
		preset: &domainpreset.Preset{ID: presetID, Width: 1080, Height: 1920, CropMode: "face_track", VideoCodec: "h264", AudioCodec: "aac"},
	}
	media := &fakeMediaWrite{mediaID: mediaID, source: "sources/vid.mp4", proxyKey: "proxies/vid.mp4"}
	jobs := &fakeJobs{}

	h := NewCreateHandler(renders, timelines, presets, media, nil, jobs, nil, nil, "")
	got, err := h.Handle(ctx, CreateCommand{ProjectID: projectID, PresetID: presetID, Kind: string(domainrender.KindMaster)})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if got.Status != domainrender.StatusPending {
		t.Fatalf("status = %q, want pending", got.Status)
	}
	if len(jobs.reframe) != 1 {
		t.Fatalf("PublishReframeJob calls = %d, want 1", len(jobs.reframe))
	}
	if jobs.reframe[0].w != 1080 || jobs.reframe[0].h != 1920 {
		t.Fatalf("reframe target = %dx%d, want 1080x1920", jobs.reframe[0].w, jobs.reframe[0].h)
	}
	if jobs.reframe[0].mediaID != mediaID.String() || jobs.reframe[0].proxy != "proxies/vid.mp4" {
		t.Fatalf("unexpected reframe args: %+v", jobs.reframe[0])
	}
	if jobs.render != 0 {
		t.Fatalf("PublishRenderJob calls = %d, want 0", jobs.render)
	}
	var plan map[string]any
	if err := json.Unmarshal(renders.saved.Plan, &plan); err != nil {
		t.Fatalf("plan json: %v", err)
	}
	if plan["awaitingReframe"] != true {
		t.Fatalf("plan awaitingReframe = %v, want true", plan["awaitingReframe"])
	}
	if plan["mediaId"] != mediaID.String() {
		t.Fatalf("plan mediaId = %v", plan["mediaId"])
	}
}
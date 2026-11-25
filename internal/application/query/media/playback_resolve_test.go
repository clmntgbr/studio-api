package media

import (
	"context"
	"testing"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type fakeMediaPlayback struct {
	file  *domainmedia.MediaFile
	proxy *domainmedia.Asset
}

func (f *fakeMediaPlayback) WithTransaction(context.Context, func(context.Context) error) error {
	panic("unused")
}
func (f *fakeMediaPlayback) Save(context.Context, *domainmedia.MediaFile) error { panic("unused") }
func (f *fakeMediaPlayback) Update(context.Context, *domainmedia.MediaFile) error {
	panic("unused")
}
func (f *fakeMediaPlayback) UpdateFromStatuses(context.Context, *domainmedia.MediaFile, []domainmedia.Status) (bool, error) {
	panic("unused")
}
func (f *fakeMediaPlayback) GetByID(context.Context, uuid.UUID) (*domainmedia.MediaFile, error) {
	return f.file, nil
}
func (f *fakeMediaPlayback) Delete(context.Context, uuid.UUID) error { panic("unused") }
func (f *fakeMediaPlayback) SaveAsset(context.Context, *domainmedia.Asset) error {
	panic("unused")
}
func (f *fakeMediaPlayback) SaveConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaPlayback) GetConfiguration(context.Context, uuid.UUID) (*domainmedia.Configuration, error) {
	panic("unused")
}
func (f *fakeMediaPlayback) UpdateConfiguration(context.Context, *domainmedia.Configuration) error {
	panic("unused")
}
func (f *fakeMediaPlayback) GetAssetByKind(_ context.Context, _ uuid.UUID, kind string) (*domainmedia.Asset, error) {
	if kind == domainmedia.AssetKindProxy {
		return f.proxy, nil
	}
	return nil, nil
}
func (f *fakeMediaPlayback) SaveStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeMediaPlayback) UpdateStep(context.Context, *domainmedia.ProcessingStep) error {
	panic("unused")
}
func (f *fakeMediaPlayback) GetStep(context.Context, uuid.UUID, domainmedia.StepKind) (*domainmedia.ProcessingStep, error) {
	panic("unused")
}
func (f *fakeMediaPlayback) SaveStepRun(context.Context, *domainmedia.ProcessingStepRun) error {
	panic("unused")
}

func TestResolvePlaybackObject_autoFallsBackToSource(t *testing.T) {
	id := uuid.New()
	src := "media/" + id.String() + "/source/orig.mp4"
	repo := &fakeMediaPlayback{
		file: &domainmedia.MediaFile{ID: id, StorageKey: &src},
	}
	key, kind, err := resolvePlaybackObject(context.Background(), repo, id, "auto")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if kind != "source" || key != src {
		t.Fatalf("got kind=%q key=%q", kind, key)
	}
}

func TestResolvePlaybackObject_autoPrefersProxy(t *testing.T) {
	id := uuid.New()
	src := "media/" + id.String() + "/source/orig.mp4"
	repo := &fakeMediaPlayback{
		file:  &domainmedia.MediaFile{ID: id, StorageKey: &src},
		proxy: &domainmedia.Asset{StorageKey: "media/" + id.String() + "/proxy/proxy_720p.mp4"},
	}
	key, kind, err := resolvePlaybackObject(context.Background(), repo, id, "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if kind != "proxy" || key != repo.proxy.StorageKey {
		t.Fatalf("got kind=%q key=%q", kind, key)
	}
}

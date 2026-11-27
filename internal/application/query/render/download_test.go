package render

import (
	"testing"

	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

func TestCollectDownloadKeys(t *testing.T) {
	key := "projects/x/renders/y/out.mp4"
	r := &domainrender.Render{
		ID:          uuid.New(),
		StorageKey:  &key,
		OutputParts: []string{"projects/x/renders/y/part-01.mp4", "projects/x/renders/y/part-02.mp4"},
	}
	got := collectDownloadKeys(r)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %v", len(got), got)
	}
}

func TestCollectDownloadKeys_MasterOnly(t *testing.T) {
	key := "out.mp4"
	r := &domainrender.Render{StorageKey: &key}
	got := collectDownloadKeys(r)
	if len(got) != 1 || got[0] != key {
		t.Fatalf("got=%v", got)
	}
}

func TestZipEntryName_Dedup(t *testing.T) {
	used := map[string]int{}
	a := zipEntryName("a/out.mp4", used)
	b := zipEntryName("b/out.mp4", used)
	if a != "out.mp4" {
		t.Fatalf("a=%q", a)
	}
	if b != "out-2.mp4" {
		t.Fatalf("b=%q", b)
	}
}

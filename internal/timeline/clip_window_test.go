package timeline

import (
	"testing"

	"github.com/google/uuid"
)

func TestContinuousClip(t *testing.T) {
	media := uuid.New()
	got := ContinuousClip(media, 3000, 48000)
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].SourceStartMs != 3000 || got[0].SourceEndMs != 48000 || got[0].OutputStartMs != 0 || got[0].OutputEndMs != 45000 {
		t.Fatalf("%+v", got[0])
	}
	if ContinuousClip(media, 5, 5) != nil {
		t.Fatal("expected nil for empty range")
	}
}

func TestClipWindow_IntersectsAndRebases(t *testing.T) {
	media := uuid.New()
	other := uuid.New()
	segs := []Segment{
		{MediaFileID: media, SourceStartMs: 0, SourceEndMs: 5000, OutputStartMs: 0, OutputEndMs: 5000},
		{MediaFileID: other, SourceStartMs: 0, SourceEndMs: 2000, OutputStartMs: 5000, OutputEndMs: 7000},
		{MediaFileID: media, SourceStartMs: 8000, SourceEndMs: 12000, OutputStartMs: 7000, OutputEndMs: 11000},
	}
	got := ClipWindow(segs, media, 3000, 10000)
	if len(got) != 2 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[0].SourceStartMs != 3000 || got[0].SourceEndMs != 5000 || got[0].OutputStartMs != 0 || got[0].OutputEndMs != 2000 {
		t.Fatalf("first=%+v", got[0])
	}
	if got[1].SourceStartMs != 8000 || got[1].SourceEndMs != 10000 || got[1].OutputStartMs != 2000 || got[1].OutputEndMs != 4000 {
		t.Fatalf("second=%+v", got[1])
	}
}

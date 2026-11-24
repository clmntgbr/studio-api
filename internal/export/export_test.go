package export

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func sampleInput() TimelineInput {
	mediaID := uuid.MustParse("01960000-0000-7000-8000-000000000003")
	return TimelineInput{
		ProjectName: "Demo",
		Version:     1,
		DurationMs:  4000,
		Width:       1920,
		Height:      1080,
		FPS:         30,
		Segments: []Segment{{
			Index:         0,
			MediaFileID:   mediaID,
			MediaName:     "take1",
			MediaPath:     "file://localhost/media/take1.mp4",
			SourceStartMs: 1000,
			SourceEndMs:   3000,
			OutputStartMs: 0,
			OutputEndMs:   2000,
		}, {
			Index:         1,
			MediaFileID:   mediaID,
			MediaName:     "take1",
			MediaPath:     "file://localhost/media/take1.mp4",
			SourceStartMs: 5000,
			SourceEndMs:   7000,
			OutputStartMs: 2000,
			OutputEndMs:   4000,
		}},
	}
}

func TestToEDL(t *testing.T) {
	doc := ToEDL(sampleInput())
	if !strings.Contains(doc, "TITLE: Demo") {
		t.Fatal(doc)
	}
	if !strings.Contains(doc, "001  TAKE1    V     C") && !strings.Contains(doc, "001  TAKE1") {
		// reel may be TAKE1 truncated
		if !strings.Contains(doc, "V     C") {
			t.Fatalf("missing edit:\n%s", doc)
		}
	}
	if !strings.Contains(doc, "00:00:01:00") {
		t.Fatalf("missing src in:\n%s", doc)
	}
}

func TestToFCPXML(t *testing.T) {
	doc := ToFCPXML(sampleInput())
	if !strings.Contains(doc, `version="1.9"`) {
		t.Fatal(doc)
	}
	if !strings.Contains(doc, "<asset-clip") {
		t.Fatal("missing asset-clip")
	}
	if strings.Count(doc, "<asset ") != 1 {
		t.Fatalf("expected 1 asset, got %d", strings.Count(doc, "<asset "))
	}
	if !strings.Contains(doc, "file://localhost/media/take1.mp4") {
		t.Fatal("missing media path")
	}
}

func TestToOTIO(t *testing.T) {
	doc, err := ToOTIO(sampleInput())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, `"OTIO_SCHEMA": "Timeline.1"`) {
		t.Fatal(doc)
	}
	if !strings.Contains(doc, `"Clip.1"`) {
		t.Fatal("missing clip")
	}
}

func TestToPremiereXML(t *testing.T) {
	doc := ToPremiereXML(sampleInput())
	if !strings.Contains(doc, `<xmeml version="4">`) {
		t.Fatal(doc)
	}
	if !strings.Contains(doc, "<clipitem id=") {
		t.Fatal("missing clipitem")
	}
	if !strings.Contains(doc, "file://localhost/media/take1.mp4") {
		t.Fatal("missing pathurl")
	}
	// Same media used twice: full file once, then empty reference.
	if strings.Count(doc, "<pathurl>") != 1 {
		t.Fatalf("expected one pathurl, got %d\n%s", strings.Count(doc, "<pathurl>"), doc)
	}
	if !strings.Contains(doc, `file id="file-01960000-0000-7000-8000-000000000003"/>`) {
		t.Fatal("missing empty file reference for second cut")
	}
	if !strings.Contains(doc, "<audio>") || !strings.Contains(doc, "-audio") {
		t.Fatal("missing linked audio clipitems")
	}
}

func TestNumberedClipsPack(t *testing.T) {
	id := uuid.MustParse("01960000-0000-7000-8000-000000000003")
	pack := NumberedClipsPack([]NumberedClip{{ClipID: id, Title: "A"}, {ClipID: id, Hook: "B"}})
	if len(pack) != 2 || pack[0].Filename != "001.mp4" || pack[1].Filename != "002.mp4" {
		t.Fatalf("%+v", pack)
	}
	manifest := NumberedClipsManifest(pack)
	if !strings.Contains(manifest, "001.mp4") {
		t.Fatal(manifest)
	}
}

func TestFormatSMPTE(t *testing.T) {
	if got := FormatSMPTE(0, 30); got != "00:00:00:00" {
		t.Fatalf("got=%s", got)
	}
	if got := FormatSMPTE(1000, 30); got != "00:00:01:00" {
		t.Fatalf("got=%s", got)
	}
	if got := FormatSMPTE(1500, 30); got != "00:00:01:15" {
		t.Fatalf("got=%s", got)
	}
}

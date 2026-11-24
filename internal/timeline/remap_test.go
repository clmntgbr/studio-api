package timeline

import (
	"testing"

	"github.com/google/uuid"
)

func sampleKeepSegments() []Segment {
	mediaID := uuid.MustParse("01960000-0000-7000-8000-000000000003")
	return []Segment{
		{
			MediaFileID: mediaID, SegmentIndex: 0,
			SourceStartMs: 0, SourceEndMs: 2200,
			OutputStartMs: 0, OutputEndMs: 2200,
		},
		{
			MediaFileID: mediaID, SegmentIndex: 1,
			SourceStartMs: 4900, SourceEndMs: 10_000,
			OutputStartMs: 2200, OutputEndMs: 7300,
		},
	}
}

func TestSourceToOutput_InsideKeep(t *testing.T) {
	segs := sampleKeepSegments()
	out, ok := SourceToOutput(1000, segs)
	if !ok || out != 1000 {
		t.Fatalf("got %d ok=%v", out, ok)
	}
	out, ok = SourceToOutput(5000, segs)
	if !ok || out != 2300 { // 2200 + (5000-4900)
		t.Fatalf("got %d ok=%v", out, ok)
	}
}

func TestSourceToOutput_InCut(t *testing.T) {
	_, ok := SourceToOutput(3000, sampleKeepSegments())
	if ok {
		t.Fatal("expected cut region")
	}
}

func TestSourceToOutput_Boundaries(t *testing.T) {
	segs := sampleKeepSegments()
	out, ok := SourceToOutput(0, segs)
	if !ok || out != 0 {
		t.Fatalf("start %d ok=%v", out, ok)
	}
	out, ok = SourceToOutput(2200, segs)
	if !ok || out != 2200 {
		t.Fatalf("seg0 end %d ok=%v", out, ok)
	}
	out, ok = SourceToOutput(4900, segs)
	if !ok || out != 2200 {
		t.Fatalf("seg1 start (splice) %d ok=%v", out, ok)
	}
	out, ok = SourceToOutput(10_000, segs)
	if !ok || out != 7300 {
		t.Fatalf("media end %d ok=%v", out, ok)
	}
}

func TestOutputToSource_SplicePrefersNextKeep(t *testing.T) {
	segs := sampleKeepSegments()
	src, ok := OutputToSource(2200, segs)
	if !ok || src != 4900 {
		t.Fatalf("splice got %d ok=%v want 4900", src, ok)
	}
}

func TestOutputToSource_RoundTripInterior(t *testing.T) {
	segs := sampleKeepSegments()
	// Exclude the ambiguous splice endpoints 2200 and 4900 (same output time).
	for _, src := range []int64{0, 500, 2199, 5000, 6000, 9999} {
		out, ok := SourceToOutput(src, segs)
		if !ok {
			t.Fatalf("source %d not mappable", src)
		}
		back, ok := OutputToSource(out, segs)
		if !ok || back != src {
			t.Fatalf("roundtrip src=%d out=%d back=%d ok=%v", src, out, back, ok)
		}
	}
	out, ok := SourceToOutput(10_000, segs)
	if !ok || out != 7300 {
		t.Fatalf("end source→out %d ok=%v", out, ok)
	}
	back, ok := OutputToSource(7300, segs)
	if !ok || back != 10_000 {
		t.Fatalf("end out→source %d ok=%v", back, ok)
	}
}

func TestOutputToSource_Outside(t *testing.T) {
	_, ok := OutputToSource(8000, sampleKeepSegments())
	if ok {
		t.Fatal("expected outside")
	}
	_, ok = OutputToSource(-1, sampleKeepSegments())
	if ok {
		t.Fatal("expected negative reject")
	}
}

func TestRemap_EmptySegments(t *testing.T) {
	if _, ok := SourceToOutput(0, nil); ok {
		t.Fatal("empty source")
	}
	if _, ok := OutputToSource(0, nil); ok {
		t.Fatal("empty output")
	}
}

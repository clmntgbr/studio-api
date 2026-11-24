package transcript

import (
	"strings"
	"testing"
)

func TestToSRT_Empty(t *testing.T) {
	if ToSRT(nil) != "" {
		t.Fatal("expected empty")
	}
}

func TestToSRT_GroupsByGap(t *testing.T) {
	out := ToSRT([]Word{
		{Text: "bonjour", SourceStartMs: 0, SourceEndMs: 400},
		{Text: "monde", SourceStartMs: 450, SourceEndMs: 900},
		{Text: "au", SourceStartMs: 3000, SourceEndMs: 3200},
		{Text: "revoir", SourceStartMs: 3250, SourceEndMs: 3600},
	})
	if !strings.Contains(out, "bonjour monde") {
		t.Fatalf("cue1 missing: %q", out)
	}
	if !strings.Contains(out, "au revoir") {
		t.Fatalf("cue2 missing: %q", out)
	}
	if !strings.Contains(out, "00:00:00,000 --> 00:00:00,900") {
		t.Fatalf("timing1: %q", out)
	}
	if !strings.Contains(out, "00:00:03,000 --> 00:00:03,600") {
		t.Fatalf("timing2: %q", out)
	}
}

func TestFormatSRTTime(t *testing.T) {
	if got := formatSRTTime(3661001); got != "01:01:01,001" {
		t.Fatalf("got %s", got)
	}
}

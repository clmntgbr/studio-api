package timeline

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuild_SilencePaddingAndKeep(t *testing.T) {
	mediaID := uuid.New()
	detID := uuid.New()
	out := Build(Input{
		MediaID:    mediaID,
		DurationMs: 10_000,
		Detections: []Detection{{
			ID: detID, Kind: KindSilence, SourceStartMs: 2000, SourceEndMs: 5000,
		}},
		Config: Config{
			SilenceEnabled:       true,
			SilenceMinDurationMs: 500,
			PaddingBeforeMs:      100,
			PaddingAfterMs:       200,
			SpeechMinDurationMs:  100,
		},
	})
	if len(out.Decisions) != 1 || out.Decisions[0].Action != ActionRemove {
		t.Fatalf("decision=%+v", out.Decisions)
	}
	// silence shrinks to 2200..4900 → keeps 0..2200 and 4900..10000
	if len(out.Segments) != 2 {
		t.Fatalf("segments=%d %#v", len(out.Segments), out.Segments)
	}
	if out.Segments[0].SourceStartMs != 0 || out.Segments[0].SourceEndMs != 2200 {
		t.Fatalf("seg0=%+v", out.Segments[0])
	}
	if out.Segments[1].SourceStartMs != 4900 || out.Segments[1].SourceEndMs != 10_000 {
		t.Fatalf("seg1=%+v", out.Segments[1])
	}
	if out.DurationMs != 2200+(10_000-4900) {
		t.Fatalf("duration=%d", out.DurationMs)
	}
	if out.Fingerprint == "" {
		t.Fatal("empty fingerprint")
	}
}

func TestBuild_KeepOverridePreservesSilence(t *testing.T) {
	mediaID := uuid.New()
	detID := uuid.New()
	kind := KindSilence
	out := Build(Input{
		MediaID:    mediaID,
		DurationMs: 5000,
		Detections: []Detection{{
			ID: detID, Kind: KindSilence, SourceStartMs: 1000, SourceEndMs: 3000,
		}},
		Overrides: []Override{{
			ID: uuid.New(), Kind: &kind, Action: ActionKeep, SourceStartMs: 1000, SourceEndMs: 3000,
		}},
		Config: Config{SilenceEnabled: true, SilenceMinDurationMs: 100, SpeechMinDurationMs: 50},
	})
	if out.Decisions[0].Action != ActionKeep || out.Decisions[0].Origin != "user" {
		t.Fatalf("decision=%+v", out.Decisions[0])
	}
	if len(out.Segments) != 1 || out.Segments[0].SourceStartMs != 0 || out.Segments[0].SourceEndMs != 5000 {
		t.Fatalf("segments=%#v", out.Segments)
	}
}

func TestBuild_KeepOverrideSurvivesShiftedDetection(t *testing.T) {
	// After a threshold re-run, detection bounds move slightly; keep still matches via ≥50% overlap.
	kind := KindFiller
	out := Build(Input{
		MediaID:    uuid.New(),
		DurationMs: 5000,
		Detections: []Detection{{
			ID: uuid.New(), Kind: KindFiller, SourceStartMs: 1100, SourceEndMs: 2900,
		}},
		Overrides: []Override{{
			ID: uuid.New(), Kind: &kind, Action: ActionKeep, SourceStartMs: 1000, SourceEndMs: 3000,
		}},
		Config: Config{SilenceEnabled: false, SpeechMinDurationMs: 50},
	})
	if len(out.Decisions) != 1 || out.Decisions[0].Action != ActionKeep || out.Decisions[0].Origin != "user" {
		t.Fatalf("decision=%+v", out.Decisions)
	}
}

func TestBuild_ShortSilenceIgnored(t *testing.T) {
	out := Build(Input{
		MediaID:    uuid.New(),
		DurationMs: 3000,
		Detections: []Detection{{
			ID: uuid.New(), Kind: KindSilence, SourceStartMs: 1000, SourceEndMs: 1200,
		}},
		Config: Config{SilenceEnabled: true, SilenceMinDurationMs: 500, SpeechMinDurationMs: 50},
	})
	if len(out.Segments) != 1 || out.DurationMs != 3000 {
		t.Fatalf("segments=%#v dur=%d", out.Segments, out.DurationMs)
	}
}

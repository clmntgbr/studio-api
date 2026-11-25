package llm

import (
	"context"
	"testing"
)

func TestResolveBounds_SnapsToWords(t *testing.T) {
	words := []Word{
		{Text: "Hello", SourceStartMs: 0, SourceEndMs: 200},
		{Text: "world.", SourceStartMs: 200, SourceEndMs: 500},
		{Text: "Next", SourceStartMs: 600, SourceEndMs: 800},
	}
	first, last, ok := ResolveBounds(words, 50, 450)
	if !ok || first != 0 || last != 1 {
		t.Fatalf("first=%d last=%d ok=%v", first, last, ok)
	}
}

func TestExpandClipBounds_ReachesMinDuration(t *testing.T) {
	words := make([]Word, 0, 100)
	for i := 0; i < 100; i++ {
		start := int64(i * 1000)
		words = append(words, Word{Text: "w", SourceStartMs: start, SourceEndMs: start + 900})
	}
	// 5s window → expand toward ~25s+
	first, last, ok := ResolveBounds(words, 10_000, 15_000)
	if !ok {
		t.Fatal("resolve")
	}
	first, last = ExpandClipBounds(words, first, last)
	dur := words[last].SourceEndMs - words[first].SourceStartMs
	if dur < ViralClipMinMs {
		t.Fatalf("dur=%d want >= %d (first=%d last=%d)", dur, ViralClipMinMs, first, last)
	}
	if dur > ViralClipMaxMs {
		t.Fatalf("dur=%d want <= %d", dur, ViralClipMaxMs)
	}
}

func TestMockAnalyzer_SponsorKeyword(t *testing.T) {
	words := []Word{
		{Text: "Merci", SourceStartMs: 0, SourceEndMs: 100},
		{Text: "à", SourceStartMs: 100, SourceEndMs: 150},
		{Text: "notre", SourceStartMs: 150, SourceEndMs: 250},
		{Text: "sponsor", SourceStartMs: 250, SourceEndMs: 400},
		{Text: "NordVPN", SourceStartMs: 400, SourceEndMs: 700},
		{Text: "aujourd'hui.", SourceStartMs: 700, SourceEndMs: 1100},
	}
	res, err := NewMockAnalyzer().Analyze(context.Background(), words, []string{"sponsors"}, "fr")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Sponsors) == 0 {
		t.Fatal("expected sponsor hit")
	}
}

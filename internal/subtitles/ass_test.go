package subtitles

import (
	"strings"
	"testing"

	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

func TestRemapWords_DropsCutRange(t *testing.T) {
	mediaID := uuid.New()
	segs := []eng.Segment{
		{MediaFileID: mediaID, SourceStartMs: 0, SourceEndMs: 1000, OutputStartMs: 0, OutputEndMs: 1000},
		{MediaFileID: mediaID, SourceStartMs: 3000, SourceEndMs: 5000, OutputStartMs: 1000, OutputEndMs: 3000},
	}
	words := []Word{
		{MediaFileID: mediaID, Text: "hello", SourceStartMs: 100, SourceEndMs: 400},
		{MediaFileID: mediaID, Text: "cut", SourceStartMs: 1500, SourceEndMs: 2000},
		{MediaFileID: mediaID, Text: "world", SourceStartMs: 3200, SourceEndMs: 3600},
	}
	out := RemapWords(words, segs)
	if len(out) != 2 {
		t.Fatalf("len=%d %#v", len(out), out)
	}
	if out[0].Text != "hello" || out[0].OutputStartMs != 100 || out[0].OutputEndMs != 400 {
		t.Fatalf("first=%+v", out[0])
	}
	if out[1].Text != "world" || out[1].OutputStartMs != 1200 || out[1].OutputEndMs != 1600 {
		t.Fatalf("second=%+v", out[1])
	}
}

func TestRemapWords_TruncatesStraddle(t *testing.T) {
	mediaID := uuid.New()
	segs := []eng.Segment{
		{MediaFileID: mediaID, SourceStartMs: 0, SourceEndMs: 1000, OutputStartMs: 0, OutputEndMs: 1000},
	}
	words := []Word{{MediaFileID: mediaID, Text: "edge", SourceStartMs: 800, SourceEndMs: 1500}}
	out := RemapWords(words, segs)
	if len(out) != 1 || out[0].OutputStartMs != 800 || out[0].OutputEndMs != 1000 {
		t.Fatalf("out=%+v", out)
	}
}

func TestRemapWords_ScopesByMedia(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	segs := []eng.Segment{
		{MediaFileID: a, SourceStartMs: 0, SourceEndMs: 1000, OutputStartMs: 0, OutputEndMs: 1000},
		{MediaFileID: b, SourceStartMs: 0, SourceEndMs: 1000, OutputStartMs: 1000, OutputEndMs: 2000},
	}
	words := []Word{
		{MediaFileID: b, Text: "clip-b", SourceStartMs: 100, SourceEndMs: 300},
		{MediaFileID: a, Text: "clip-a", SourceStartMs: 100, SourceEndMs: 300},
	}
	out := RemapWords(words, segs)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Text != "clip-a" || out[0].OutputStartMs != 100 {
		t.Fatalf("first=%+v", out[0])
	}
	if out[1].Text != "clip-b" || out[1].OutputStartMs != 1100 {
		t.Fatalf("second=%+v", out[1])
	}
}

func TestGenerateASS_PopResetsScale(t *testing.T) {
	style := DefaultStyle()
	style.Animation = "pop"
	style.Uppercase = false
	style.WordsPerLine = 2
	words := []TimedWord{
		{Text: "one", OutputStartMs: 0, OutputEndMs: 200},
		{Text: "two", OutputStartMs: 200, OutputEndMs: 400},
	}
	doc := GenerateASS(words, style)
	if !strings.Contains(doc, `{\fscx100\fscy100\c`) {
		t.Fatalf("expected scale reset, got:\n%s", doc)
	}
}

func TestGenerateASS_PlayResFromStyle(t *testing.T) {
	style := DefaultStyle()
	style.PlayResX = 1080
	style.PlayResY = 1920
	doc := GenerateASS(nil, style)
	if !strings.Contains(doc, "PlayResX: 1080") || !strings.Contains(doc, "PlayResY: 1920") {
		t.Fatalf("doc=%s", doc)
	}
}

func TestGenerateASS_KaraokeHighlight(t *testing.T) {
	style := DefaultStyle()
	style.Animation = "none"
	style.Uppercase = false
	style.WordsPerLine = 3
	words := []TimedWord{
		{Text: "bonjour", OutputStartMs: 0, OutputEndMs: 400},
		{Text: "le", OutputStartMs: 400, OutputEndMs: 600},
		{Text: "monde", OutputStartMs: 600, OutputEndMs: 1000},
	}
	doc := GenerateASS(words, style)
	if !strings.Contains(doc, "[Events]") {
		t.Fatal("missing events")
	}
	if strings.Count(doc, "Dialogue:") != 3 {
		t.Fatalf("dialogues=%d\n%s", strings.Count(doc, "Dialogue:"), doc)
	}
	if !strings.Contains(doc, toASSColor(style.HighlightColor)) {
		t.Fatal("missing highlight color")
	}
	if !strings.Contains(doc, "0:00:00.00") {
		t.Fatal("missing start time")
	}
}

func TestGenerateASS_UppercaseAndEmojiFilter(t *testing.T) {
	style := DefaultStyle()
	style.Uppercase = true
	words := []TimedWord{{Text: "hé 😀", OutputStartMs: 0, OutputEndMs: 300}}
	doc := GenerateASS(words, style)
	if !strings.Contains(doc, "HÉ") {
		t.Fatalf("expected uppercase, got:\n%s", doc)
	}
	if strings.Contains(doc, "😀") {
		t.Fatal("emoji should be filtered")
	}
}

func TestGenerateSRT_Remapped(t *testing.T) {
	style := DefaultStyle()
	style.Uppercase = false
	style.WordsPerLine = 2
	words := []TimedWord{
		{Text: "a", OutputStartMs: 0, OutputEndMs: 200},
		{Text: "b", OutputStartMs: 200, OutputEndMs: 400},
	}
	doc := GenerateSRT(words, style)
	if !strings.Contains(doc, "00:00:00,000 -->") {
		t.Fatalf("bad srt:\n%s", doc)
	}
	if !strings.Contains(doc, "a b") {
		t.Fatalf("missing text:\n%s", doc)
	}
}

func TestToASSColor(t *testing.T) {
	if got := toASSColor("#FFE600"); got != "&H0000E6FF" {
		t.Fatalf("got=%s", got)
	}
}

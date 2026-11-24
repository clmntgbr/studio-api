package transcript

import (
	"fmt"
	"strings"
	"time"
)

// Word is a timed transcript token in source milliseconds.
type Word struct {
	Text          string
	SourceStartMs int64
	SourceEndMs   int64
}

const (
	defaultGapMs      int64 = 800
	defaultMaxChars         = 42
	defaultMaxDuration int64 = 6000
)

// ToSRT builds a WebVTT-compatible SRT document from timed words.
func ToSRT(words []Word) string {
	if len(words) == 0 {
		return ""
	}
	cues := groupWords(words, defaultGapMs, defaultMaxChars, defaultMaxDuration)
	var b strings.Builder
	for i, cue := range cues {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d\n", i+1)
		fmt.Fprintf(&b, "%s --> %s\n", formatSRTTime(cue.startMs), formatSRTTime(cue.endMs))
		b.WriteString(cue.text)
		b.WriteByte('\n')
	}
	return b.String()
}

type cue struct {
	startMs int64
	endMs   int64
	text    string
}

func groupWords(words []Word, gapMs int64, maxChars int, maxDurationMs int64) []cue {
	out := make([]cue, 0)
	var start, end int64
	var parts []string
	flush := func() {
		if len(parts) == 0 {
			return
		}
		out = append(out, cue{startMs: start, endMs: end, text: strings.Join(parts, " ")})
		parts = nil
	}
	for _, w := range words {
		if w.Text == "" || w.SourceEndMs < w.SourceStartMs {
			continue
		}
		if len(parts) == 0 {
			start, end = w.SourceStartMs, w.SourceEndMs
			parts = []string{w.Text}
			continue
		}
		nextText := strings.Join(append(append([]string{}, parts...), w.Text), " ")
		gap := w.SourceStartMs - end
		dur := w.SourceEndMs - start
		if gap > gapMs || len(nextText) > maxChars || dur > maxDurationMs {
			flush()
			start, end = w.SourceStartMs, w.SourceEndMs
			parts = []string{w.Text}
			continue
		}
		parts = append(parts, w.Text)
		end = w.SourceEndMs
	}
	flush()
	return out
}

func formatSRTTime(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	d := time.Duration(ms) * time.Millisecond
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	milli := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, milli)
}

package llm

import (
	"context"
	"strings"
	"unicode"
)

// MockAnalyzer produces deterministic heuristic hits without calling a remote LLM.
// Used when LLM_PROVIDER=mock (default for local/dev).
type MockAnalyzer struct{}

func NewMockAnalyzer() *MockAnalyzer { return &MockAnalyzer{} }

var sponsorKeywords = []string{
	"sponsor", "sponsors", "partenariat", "partenaire", "pub", "publicité",
	"code promo", "lien en description", "merci à", "proposé par",
	"brought to you", "this episode is sponsored", "use code",
}

func (a *MockAnalyzer) Analyze(_ context.Context, words []Word, tasks []string, _ string) (*Result, error) {
	want := taskSet(tasks)
	out := &Result{Provider: "mock", Sponsors: []SponsorHit{}, Chapters: []ChapterHit{}, Clips: []ClipHit{}}
	if len(words) == 0 {
		return out, nil
	}
	if want["sponsors"] {
		out.Sponsors = a.findSponsors(words)
	}
	if want["chapters"] {
		out.Chapters = a.buildChapters(words)
	}
	if want["viral"] {
		out.Clips = a.buildClips(words)
	}
	return out, nil
}

func (a *MockAnalyzer) findSponsors(words []Word) []SponsorHit {
	hits := make([]SponsorHit, 0)
	n := len(words)
	for i := 0; i < n; i++ {
		window := normalizeJoin(words, i, min(i+6, n))
		matched := ""
		for _, kw := range sponsorKeywords {
			if strings.Contains(window, kw) {
				matched = kw
				break
			}
		}
		if matched == "" {
			continue
		}
		end := min(i+12, n-1)
		first, last, ok := ResolveBounds(words, words[i].SourceStartMs, words[end].SourceEndMs)
		if !ok {
			continue
		}
		conf := 0.72
		fi, li := first, last
		hits = append(hits, SponsorHit{
			SourceStartMs: words[first].SourceStartMs,
			SourceEndMs:   words[last].SourceEndMs,
			FirstWord:     &fi,
			LastWord:      &li,
			Confidence:    conf,
			Reason:        "keyword:" + matched,
		})
		i = last
	}
	return dedupeSponsors(hits)
}

func (a *MockAnalyzer) buildChapters(words []Word) []ChapterHit {
	const chunkMs int64 = 180_000
	start := words[0].SourceStartMs
	endLimit := words[len(words)-1].SourceEndMs
	chapters := make([]ChapterHit, 0)
	idx := 1
	for t := start; t < endLimit; t += chunkMs {
		segEnd := t + chunkMs
		if segEnd > endLimit {
			segEnd = endLimit
		}
		first, last, ok := ResolveBounds(words, t, segEnd)
		if !ok {
			continue
		}
		title := "Chapter " + itoa(idx)
		if last >= first {
			rs := []rune(strings.TrimSpace(words[first].Text))
			if len(rs) > 0 {
				title = string(unicode.ToUpper(rs[0])) + string(rs[1:])
				if len(rs) > 40 {
					title = string(rs[:40]) + "…"
				}
			}
		}
		chapters = append(chapters, ChapterHit{
			Title:         title,
			SourceStartMs: words[first].SourceStartMs,
			SourceEndMs:   words[last].SourceEndMs,
		})
		idx++
		if idx > 24 {
			break
		}
	}
	return chapters
}

func (a *MockAnalyzer) buildClips(words []Word) []ClipHit {
	if len(words) < 8 {
		return nil
	}
	mid := len(words) / 3
	end := min(mid+40, len(words)-1)
	first, last, ok := ResolveBounds(words, words[mid].SourceStartMs, words[end].SourceEndMs)
	if !ok {
		return nil
	}
	score := 0.61
	hook := words[first].Text
	title := truncateRunes("Highlight: "+hook, 80)
	return []ClipHit{{
		SourceStartMs: words[first].SourceStartMs,
		SourceEndMs:   words[last].SourceEndMs,
		Score:         score,
		Hook:          hook,
		Reason:        "heuristic mid-video highlight",
		Title:         title,
		Description:   "Auto-generated clip from the mid section of the transcript.",
		Hashtags:      []string{"#shorts", "#clip", "#highlight"},
	}}
}

func taskSet(tasks []string) map[string]bool {
	out := map[string]bool{}
	if len(tasks) == 0 {
		out["sponsors"] = true
		out["chapters"] = true
		out["viral"] = true
		return out
	}
	for _, t := range tasks {
		out[strings.ToLower(strings.TrimSpace(t))] = true
	}
	return out
}

func normalizeJoin(words []Word, from, to int) string {
	parts := make([]string, 0, to-from)
	for i := from; i < to; i++ {
		parts = append(parts, normalize(words[i].Text))
	}
	return strings.Join(parts, " ")
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func dedupeSponsors(in []SponsorHit) []SponsorHit {
	if len(in) == 0 {
		return in
	}
	out := make([]SponsorHit, 0, len(in))
	var lastEnd int64 = -1
	for _, h := range in {
		if h.SourceStartMs < lastEnd {
			continue
		}
		out = append(out, h)
		lastEnd = h.SourceEndMs
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

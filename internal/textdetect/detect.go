package textdetect

import "strings"

type Word struct {
	Index         int
	Text          string
	SourceStartMs int64
	SourceEndMs   int64
}

type Interval struct {
	Kind          string
	SourceStartMs int64
	SourceEndMs   int64
	FirstWord     int
	LastWord      int
}

type Options struct {
	Level              string
	FillersEnabled     bool
	RepetitionsEnabled bool
	FalseStartsEnabled bool
}

const DetectorVersion = "rules.fr@1"

// Detect finds filler / repetition / false-start ranges on timed words.
func Detect(words []Word, opts Options) []Interval {
	out := make([]Interval, 0)
	if opts.FillersEnabled {
		out = append(out, detectFillers(words, opts.Level)...)
	}
	if opts.RepetitionsEnabled {
		out = append(out, detectRepetitions(words)...)
	}
	if opts.FalseStartsEnabled {
		out = append(out, detectFalseStarts(words)...)
	}
	return out
}

func detectFillers(words []Word, level string) []Interval {
	set := fillerSet(level)
	out := make([]Interval, 0)
	for i := 0; i < len(words); i++ {
		norm := normalizeToken(words[i].Text)
		if norm == "" {
			continue
		}
		if _, ok := set[norm]; ok {
			out = append(out, Interval{
				Kind: "filler", SourceStartMs: words[i].SourceStartMs, SourceEndMs: words[i].SourceEndMs,
				FirstWord: words[i].Index, LastWord: words[i].Index,
			})
			continue
		}
		// bigrams for multi-word fillers
		if i+1 < len(words) {
			pair := norm + " " + normalizeToken(words[i+1].Text)
			if _, ok := set[pair]; ok {
				out = append(out, Interval{
					Kind: "filler", SourceStartMs: words[i].SourceStartMs, SourceEndMs: words[i+1].SourceEndMs,
					FirstWord: words[i].Index, LastWord: words[i+1].Index,
				})
				i++
			}
		}
	}
	return out
}

func detectRepetitions(words []Word) []Interval {
	out := make([]Interval, 0)
	i := 0
	for i < len(words) {
		norm := normalizeToken(words[i].Text)
		if norm == "" {
			i++
			continue
		}
		j := i + 1
		for j < len(words) && normalizeToken(words[j].Text) == norm {
			j++
		}
		if j-i >= 2 {
			// keep last occurrence; mark earlier ones as repetition
			out = append(out, Interval{
				Kind:          "repetition",
				SourceStartMs: words[i].SourceStartMs,
				SourceEndMs:   words[j-2].SourceEndMs,
				FirstWord:     words[i].Index,
				LastWord:      words[j-2].Index,
			})
		}
		i = j
	}
	return out
}

func detectFalseStarts(words []Word) []Interval {
	// Heuristic: short aborted phrase (< 4 words) followed within 1.2s by a similar restart.
	out := make([]Interval, 0)
	for i := 0; i < len(words); i++ {
		for n := 2; n <= 3 && i+n < len(words); n++ {
			end := i + n - 1
			gapStart := end + 1
			if gapStart >= len(words) {
				continue
			}
			if words[gapStart].SourceStartMs-words[end].SourceEndMs > 1200 {
				continue
			}
			prefix := phrase(words[i : i+n])
			if len(prefix) < 2 {
				continue
			}
			for m := n; m <= n+1 && gapStart+m <= len(words); m++ {
				restart := phrase(words[gapStart : gapStart+m])
				if strings.HasPrefix(restart, prefix) || strings.HasPrefix(prefix, restart) {
					out = append(out, Interval{
						Kind:          "false_start",
						SourceStartMs: words[i].SourceStartMs,
						SourceEndMs:   words[end].SourceEndMs,
						FirstWord:     words[i].Index,
						LastWord:      words[end].Index,
					})
					return out // one per pass is enough for v1
				}
			}
		}
	}
	return out
}

func phrase(words []Word) string {
	parts := make([]string, 0, len(words))
	for _, w := range words {
		if n := normalizeToken(w.Text); n != "" {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, " ")
}

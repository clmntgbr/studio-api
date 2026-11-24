package llm

import (
	"strings"
	"unicode"
)

// ResolveBounds snaps a [start,end] range onto word indices, preferring sentence ends.
func ResolveBounds(words []Word, startMs, endMs int64) (first, last int, ok bool) {
	if len(words) == 0 || endMs <= startMs {
		return 0, 0, false
	}
	first = -1
	last = -1
	for i, w := range words {
		if w.SourceEndMs <= startMs {
			continue
		}
		if w.SourceStartMs >= endMs {
			break
		}
		if first < 0 {
			first = i
		}
		last = i
	}
	if first < 0 || last < 0 {
		return 0, 0, false
	}
	// Prefer extending to nearest sentence end within +3 words.
	for i := last; i < len(words) && i <= last+3; i++ {
		if isSentenceEnd(words[i].Text) {
			last = i
			break
		}
	}
	if words[last].SourceEndMs <= words[first].SourceStartMs {
		return 0, 0, false
	}
	return first, last, true
}

func isSentenceEnd(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" {
		return false
	}
	r, _ := utf8LastRune(t)
	return r == '.' || r == '!' || r == '?' || r == '…'
}

func utf8LastRune(s string) (rune, bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if unicode.IsSpace(rune(s[i])) {
			continue
		}
		// walk back for UTF-8
		for i > 0 && !utf8Start(s[i]) {
			i--
		}
		r := []rune(s[i:])
		if len(r) == 0 {
			return 0, false
		}
		return r[0], true
	}
	return 0, false
}

func utf8Start(b byte) bool {
	return b&0xC0 != 0x80
}

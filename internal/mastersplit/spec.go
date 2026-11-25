package mastersplit

import (
	"fmt"
	"strconv"
	"strings"
)

// Spec encodes how to slice a finished master for TikTok / multi-upload.
// Stored as one DB string on media_configuration.master_split:
//
//	"" | "off"              → no split
//	"parts:5"               → 5 equal parts
//	"durationMs:600000"     → parts of at most 10 minutes
type Spec struct {
	Parts      int   // >0 → equal N parts
	DurationMs int64 // >0 → max length per part
}

func Parse(raw string) (Spec, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" || s == "off" || s == "none" {
		return Spec{}, nil
	}
	if strings.HasPrefix(s, "parts:") {
		n, err := strconv.Atoi(strings.TrimSpace(s[len("parts:"):]))
		if err != nil || n < 2 || n > 50 {
			return Spec{}, fmt.Errorf("masterSplit parts must be 2–50")
		}
		return Spec{Parts: n}, nil
	}
	if strings.HasPrefix(s, "durationms:") {
		ms, err := strconv.ParseInt(strings.TrimSpace(s[len("durationms:"):]), 10, 64)
		if err != nil || ms < 60_000 || ms > 3_600_000 {
			return Spec{}, fmt.Errorf("masterSplit durationMs must be 60000–3600000")
		}
		return Spec{DurationMs: ms}, nil
	}
	return Spec{}, fmt.Errorf("masterSplit must be parts:N or durationMs:N")
}

func FormatParts(n int) string {
	return fmt.Sprintf("parts:%d", n)
}

func FormatDurationMs(ms int64) string {
	return fmt.Sprintf("durationMs:%d", ms)
}

func (s Spec) IsZero() bool {
	return s.Parts <= 0 && s.DurationMs <= 0
}

func (s Spec) String() string {
	if s.Parts > 0 {
		return FormatParts(s.Parts)
	}
	if s.DurationMs > 0 {
		return FormatDurationMs(s.DurationMs)
	}
	return ""
}

// PartWindows returns [startMs,endMs) windows covering totalMs.
func (s Spec) PartWindows(totalMs int64) [][2]int64 {
	if totalMs <= 0 || s.IsZero() {
		return nil
	}
	var maxMs int64
	var n int
	if s.Parts > 0 {
		n = s.Parts
		maxMs = (totalMs + int64(n) - 1) / int64(n)
	} else {
		maxMs = s.DurationMs
		n = int((totalMs + maxMs - 1) / maxMs)
	}
	if n < 2 {
		return nil // single file — no useful split
	}
	if n > 50 {
		n = 50
		maxMs = (totalMs + int64(n) - 1) / int64(n)
	}
	out := make([][2]int64, 0, n)
	var start int64
	for i := 0; i < n && start < totalMs; i++ {
		end := start + maxMs
		if end > totalMs {
			end = totalMs
		}
		if end <= start {
			break
		}
		out = append(out, [2]int64{start, end})
		start = end
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

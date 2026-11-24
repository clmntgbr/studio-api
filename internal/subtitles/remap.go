package subtitles

import (
	"sort"

	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

// Word is a timed transcript token in SourceTime.
type Word struct {
	MediaFileID   uuid.UUID
	Text          string
	SourceStartMs int64
	SourceEndMs   int64
}

// TimedWord is a word mapped into OutputTime.
type TimedWord struct {
	MediaFileID   uuid.UUID
	Text          string
	OutputStartMs int64
	OutputEndMs   int64
}

// RemapWords maps source words onto KEEP segments for their media file.
// Fully cut words are dropped; words straddling a cut are truncated to the keep edge.
func RemapWords(words []Word, segments []eng.Segment) []TimedWord {
	byMedia := make(map[uuid.UUID][]eng.Segment)
	for _, s := range segments {
		byMedia[s.MediaFileID] = append(byMedia[s.MediaFileID], s)
	}
	for id := range byMedia {
		segs := byMedia[id]
		sort.Slice(segs, func(i, j int) bool { return segs[i].SourceStartMs < segs[j].SourceStartMs })
		byMedia[id] = segs
	}

	out := make([]TimedWord, 0, len(words))
	for _, w := range words {
		if w.Text == "" || w.SourceEndMs <= w.SourceStartMs {
			continue
		}
		segs := byMedia[w.MediaFileID]
		if len(segs) == 0 {
			continue
		}
		start, startOK := eng.SourceToOutput(w.SourceStartMs, segs)
		end, endOK := eng.SourceToOutput(w.SourceEndMs, segs)

		if !startOK && !endOK {
			continue
		}
		if !startOK {
			start, startOK = snapToNextKeepStart(w.SourceStartMs, w.SourceEndMs, segs)
			if !startOK {
				continue
			}
		}
		if !endOK {
			end, endOK = snapToPrevKeepEnd(w.SourceStartMs, w.SourceEndMs, segs)
			if !endOK {
				continue
			}
		}
		if end <= start {
			continue
		}
		out = append(out, TimedWord{
			MediaFileID:   w.MediaFileID,
			Text:          w.Text,
			OutputStartMs: start,
			OutputEndMs:   end,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OutputStartMs == out[j].OutputStartMs {
			return out[i].OutputEndMs < out[j].OutputEndMs
		}
		return out[i].OutputStartMs < out[j].OutputStartMs
	})
	return out
}

func snapToNextKeepStart(srcStart, srcEnd int64, segments []eng.Segment) (int64, bool) {
	for _, s := range segments {
		if s.SourceEndMs <= srcStart {
			continue
		}
		if s.SourceStartMs >= srcEnd {
			return 0, false
		}
		src := s.SourceStartMs
		if srcStart > src {
			src = srcStart
		}
		return eng.SourceToOutput(src, segments)
	}
	return 0, false
}

func snapToPrevKeepEnd(srcStart, srcEnd int64, segments []eng.Segment) (int64, bool) {
	for i := len(segments) - 1; i >= 0; i-- {
		s := segments[i]
		if s.SourceStartMs >= srcEnd {
			continue
		}
		if s.SourceEndMs <= srcStart {
			return 0, false
		}
		src := s.SourceEndMs
		if srcEnd < src {
			src = srcEnd
		}
		return eng.SourceToOutput(src, segments)
	}
	return 0, false
}

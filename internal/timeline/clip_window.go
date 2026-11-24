package timeline

import "github.com/google/uuid"

// ClipWindow intersects KEEP timeline segments with a SourceTime clip range on one media.
// Returned segments are re-based so OutputStartMs starts at 0 for the clip render.
func ClipWindow(segs []Segment, mediaID uuid.UUID, clipStartMs, clipEndMs int64) []Segment {
	if clipEndMs <= clipStartMs || len(segs) == 0 {
		return nil
	}
	out := make([]Segment, 0, len(segs))
	var cursor int64
	for _, s := range segs {
		if s.MediaFileID != mediaID {
			continue
		}
		start := s.SourceStartMs
		end := s.SourceEndMs
		if end <= clipStartMs || start >= clipEndMs {
			continue
		}
		if start < clipStartMs {
			start = clipStartMs
		}
		if end > clipEndMs {
			end = clipEndMs
		}
		if end <= start {
			continue
		}
		dur := end - start
		out = append(out, Segment{
			MediaFileID:   mediaID,
			SourceStartMs: start,
			SourceEndMs:   end,
			OutputStartMs: cursor,
			OutputEndMs:   cursor + dur,
		})
		cursor += dur
	}
	return out
}

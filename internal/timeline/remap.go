package timeline

// SourceToOutput maps a source-time position onto output time using KEEP segments.
// ok is false when sourceMs falls inside a removed range (or outside the media).
//
// At a cut junction, the end of the previous KEEP and the start of the next KEEP
// share the same output time (the splice point).
func SourceToOutput(sourceMs int64, segments []Segment) (outputMs int64, ok bool) {
	if sourceMs < 0 || len(segments) == 0 {
		return 0, false
	}
	for _, s := range segments {
		if sourceMs < s.SourceStartMs {
			return 0, false
		}
		if sourceMs < s.SourceEndMs {
			return s.OutputStartMs + (sourceMs - s.SourceStartMs), true
		}
		if sourceMs == s.SourceEndMs {
			return s.OutputEndMs, true
		}
	}
	return 0, false
}

// OutputToSource maps an output-time position back onto source time.
// ok is false when outputMs is outside the reconstructed output duration.
//
// Output ranges are treated as half-open [outputStart, outputEnd) except for the
// final segment, which includes its end. That way a splice point resolves to the
// start of the next KEEP (forward continuity for scrubbing/playback).
func OutputToSource(outputMs int64, segments []Segment) (sourceMs int64, ok bool) {
	if outputMs < 0 || len(segments) == 0 {
		return 0, false
	}
	for i, s := range segments {
		last := i == len(segments)-1
		if outputMs < s.OutputStartMs {
			return 0, false
		}
		if outputMs < s.OutputEndMs || (last && outputMs == s.OutputEndMs) {
			return s.SourceStartMs + (outputMs - s.OutputStartMs), true
		}
	}
	return 0, false
}

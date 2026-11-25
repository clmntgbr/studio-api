package timeline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

type DetectionKind string

const (
	KindSilence    DetectionKind = "silence"
	KindFiller     DetectionKind = "filler"
	KindRepetition DetectionKind = "repetition"
	KindFalseStart DetectionKind = "false_start"
	KindSponsor    DetectionKind = "sponsor"
	KindAd         DetectionKind = "ad"
)

type Action string

const (
	ActionRemove  Action = "remove"
	ActionSuggest Action = "suggest"
	ActionKeep    Action = "keep"
)

type CutReason string

const (
	CutSilence CutReason = "silence"
	CutManual  CutReason = "manual"
	CutMicro   CutReason = "micro"
)

type Detection struct {
	ID            uuid.UUID
	Kind          DetectionKind
	SourceStartMs int64
	SourceEndMs   int64
	Confidence    *float64
}

type Override struct {
	ID            uuid.UUID
	Kind          *DetectionKind
	Action        Action
	SourceStartMs int64
	SourceEndMs   int64
}

type Config struct {
	SilenceEnabled       bool
	SilenceMinDurationMs int
	PaddingBeforeMs      int
	PaddingAfterMs       int
	SpeechMinDurationMs  int
	SponsorsAction       Action
	AdsAction            Action
}

type Input struct {
	MediaID    uuid.UUID
	DurationMs int64
	Detections []Detection
	Overrides  []Override
	Config     Config
}

type Decision struct {
	DetectionID uuid.UUID
	Action      Action
	Origin      string // auto | user
	Reason      *string
}

type Segment struct {
	MediaFileID         uuid.UUID
	SegmentIndex        int
	SourceStartMs       int64
	SourceEndMs         int64
	OutputStartMs       int64
	OutputEndMs         int64
	RemovedBeforeReason *CutReason
}

type Output struct {
	Segments    []Segment
	Decisions   []Decision
	DurationMs  int64
	Fingerprint string
}

type interval struct {
	start  int64
	end    int64
	reason CutReason
}

// Build is a pure timeline reconstruction (docs/docs.md — silence path first).
func Build(in Input) Output {
	cfg := in.Config
	if cfg.SilenceMinDurationMs <= 0 {
		cfg.SilenceMinDurationMs = 700
	}
	if cfg.SpeechMinDurationMs <= 0 {
		cfg.SpeechMinDurationMs = 800
	}

	decisions := make([]Decision, 0, len(in.Detections))
	var remove []interval

	for _, d := range in.Detections {
		action := autoAction(d, cfg)
		origin := "auto"
		if ov := matchingOverride(d, in.Overrides); ov != nil {
			action = ov.Action
			origin = "user"
		}
		decisions = append(decisions, Decision{
			DetectionID: d.ID,
			Action:      action,
			Origin:      origin,
		})
		if action != ActionRemove {
			continue
		}
		if d.Kind == KindSilence {
			start, end, ok := shrinkSilence(d, cfg)
			if !ok {
				continue
			}
			remove = append(remove, interval{start: start, end: end, reason: CutSilence})
			continue
		}
		remove = append(remove, interval{start: d.SourceStartMs, end: d.SourceEndMs, reason: cutReasonFor(d.Kind)})
	}

	// Manual removes without a matching detection.
	for _, ov := range in.Overrides {
		if ov.Action != ActionRemove {
			continue
		}
		if coversDetection(ov, in.Detections) {
			continue
		}
		remove = append(remove, interval{start: ov.SourceStartMs, end: ov.SourceEndMs, reason: CutManual})
	}

	// keep overrides without kind protect ranges from removal.
	protected := make([]interval, 0)
	for _, ov := range in.Overrides {
		if ov.Action == ActionKeep && ov.Kind == nil {
			protected = append(protected, interval{start: ov.SourceStartMs, end: ov.SourceEndMs})
		}
	}
	remove = subtractProtected(mergeIntervals(remove), protected)

	keeps := complement(in.DurationMs, remove)
	keeps = dropMicroAndMerge(keeps, cfg.SpeechMinDurationMs)

	segments := make([]Segment, 0, len(keeps))
	var outCursor int64
	var prevEnd int64
	var prevReason *CutReason
	for i, k := range keeps {
		if i > 0 {
			r := findRemoveReason(remove, prevEnd, k.start)
			prevReason = &r
		} else if k.start > 0 {
			r := findRemoveReason(remove, 0, k.start)
			prevReason = &r
		} else {
			prevReason = nil
		}
		dur := k.end - k.start
		seg := Segment{
			MediaFileID:         in.MediaID,
			SegmentIndex:        i,
			SourceStartMs:       k.start,
			SourceEndMs:         k.end,
			OutputStartMs:       outCursor,
			OutputEndMs:         outCursor + dur,
			RemovedBeforeReason: prevReason,
		}
		segments = append(segments, seg)
		outCursor += dur
		prevEnd = k.end
		prevReason = nil
	}

	return Output{
		Segments:    segments,
		Decisions:   decisions,
		DurationMs:  outCursor,
		Fingerprint: fingerprint(in, decisions, segments),
	}
}

func autoAction(d Detection, cfg Config) Action {
	switch d.Kind {
	case KindSilence:
		if !cfg.SilenceEnabled {
			return ActionKeep
		}
		return ActionRemove
	case KindSponsor:
		if cfg.SponsorsAction != "" {
			return cfg.SponsorsAction
		}
		return ActionSuggest
	case KindAd:
		if cfg.AdsAction != "" {
			return cfg.AdsAction
		}
		return ActionSuggest
	default:
		return ActionRemove
	}
}

func shrinkSilence(d Detection, cfg Config) (int64, int64, bool) {
	if d.SourceEndMs-d.SourceStartMs < int64(cfg.SilenceMinDurationMs) {
		return 0, 0, false
	}
	// Shorten: padding_after at start, padding_before at end (docs).
	start := d.SourceStartMs + int64(cfg.PaddingAfterMs)
	end := d.SourceEndMs - int64(cfg.PaddingBeforeMs)
	if end <= start {
		return 0, 0, false
	}
	return start, end, true
}

func matchingOverride(d Detection, overrides []Override) *Override {
	bestOverlap := int64(0)
	var best *Override
	detDur := d.SourceEndMs - d.SourceStartMs
	if detDur <= 0 {
		return nil
	}
	for i := range overrides {
		ov := &overrides[i]
		if ov.Kind != nil && *ov.Kind != d.Kind {
			continue
		}
		overlap := overlapMs(d.SourceStartMs, d.SourceEndMs, ov.SourceStartMs, ov.SourceEndMs)
		if overlap*2 >= detDur && overlap > bestOverlap { // ≥ 50%
			bestOverlap = overlap
			best = ov
		}
	}
	return best
}

func coversDetection(ov Override, detections []Detection) bool {
	for _, d := range detections {
		if ov.Kind != nil && *ov.Kind != d.Kind {
			continue
		}
		detDur := d.SourceEndMs - d.SourceStartMs
		if detDur <= 0 {
			continue
		}
		if overlapMs(d.SourceStartMs, d.SourceEndMs, ov.SourceStartMs, ov.SourceEndMs)*2 >= detDur {
			return true
		}
	}
	return false
}

func overlapMs(a0, a1, b0, b1 int64) int64 {
	start := a0
	if b0 > start {
		start = b0
	}
	end := a1
	if b1 < end {
		end = b1
	}
	if end <= start {
		return 0
	}
	return end - start
}

func mergeIntervals(in []interval) []interval {
	if len(in) == 0 {
		return nil
	}
	sort.Slice(in, func(i, j int) bool {
		if in[i].start == in[j].start {
			return in[i].end < in[j].end
		}
		return in[i].start < in[j].start
	})
	out := []interval{in[0]}
	for _, cur := range in[1:] {
		last := &out[len(out)-1]
		if cur.start <= last.end {
			if cur.end > last.end {
				last.end = cur.end
			}
			continue
		}
		out = append(out, cur)
	}
	return out
}

func subtractProtected(remove, protected []interval) []interval {
	if len(protected) == 0 {
		return remove
	}
	protected = mergeIntervals(protected)
	var out []interval
	for _, r := range remove {
		chunks := []interval{r}
		for _, p := range protected {
			var next []interval
			for _, c := range chunks {
				next = append(next, subtractOne(c, p)...)
			}
			chunks = next
		}
		out = append(out, chunks...)
	}
	return mergeIntervals(out)
}

func subtractOne(a, p interval) []interval {
	if p.end <= a.start || p.start >= a.end {
		return []interval{a}
	}
	var out []interval
	if p.start > a.start {
		out = append(out, interval{start: a.start, end: p.start, reason: a.reason})
	}
	if p.end < a.end {
		out = append(out, interval{start: p.end, end: a.end, reason: a.reason})
	}
	return out
}

func complement(durationMs int64, remove []interval) []interval {
	if durationMs <= 0 {
		return nil
	}
	remove = mergeIntervals(remove)
	var keeps []interval
	cursor := int64(0)
	for _, r := range remove {
		if r.start > cursor {
			keeps = append(keeps, interval{start: cursor, end: min64(r.start, durationMs)})
		}
		if r.end > cursor {
			cursor = r.end
		}
	}
	if cursor < durationMs {
		keeps = append(keeps, interval{start: cursor, end: durationMs})
	}
	return keeps
}

func dropMicroAndMerge(keeps []interval, speechMinMs int) []interval {
	if len(keeps) == 0 {
		return nil
	}
	minDur := int64(speechMinMs)
	filtered := make([]interval, 0, len(keeps))
	for _, k := range keeps {
		if k.end-k.start < minDur {
			continue
		}
		filtered = append(filtered, k)
	}
	if len(filtered) == 0 {
		return nil
	}
	out := []interval{filtered[0]}
	for _, cur := range filtered[1:] {
		last := &out[len(out)-1]
		// Merge keep islands separated by short residual gaps (micro-silence leftovers).
		if cur.start-last.end < 400 {
			last.end = cur.end
			continue
		}
		out = append(out, cur)
	}
	return out
}

func findRemoveReason(remove []interval, from, to int64) CutReason {
	for _, r := range remove {
		if overlapMs(from, to, r.start, r.end) > 0 {
			return r.reason
		}
	}
	return CutSilence
}

func cutReasonFor(kind DetectionKind) CutReason {
	switch kind {
	case KindSilence:
		return CutSilence
	default:
		return CutReason(kind)
	}
}

func fingerprint(in Input, decisions []Decision, segments []Segment) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "media=%s|dur=%d|", in.MediaID, in.DurationMs)
	_, _ = fmt.Fprintf(h, "cfg=%v|%d|%d|%d|%d|", in.Config.SilenceEnabled, in.Config.SilenceMinDurationMs, in.Config.PaddingBeforeMs, in.Config.PaddingAfterMs, in.Config.SpeechMinDurationMs)
	dets := append([]Detection(nil), in.Detections...)
	sort.Slice(dets, func(i, j int) bool { return dets[i].ID.String() < dets[j].ID.String() })
	for _, d := range dets {
		_, _ = fmt.Fprintf(h, "d=%s:%s:%d:%d|", d.ID, d.Kind, d.SourceStartMs, d.SourceEndMs)
	}
	ovs := append([]Override(nil), in.Overrides...)
	sort.Slice(ovs, func(i, j int) bool { return ovs[i].ID.String() < ovs[j].ID.String() })
	for _, o := range ovs {
		_, _ = fmt.Fprintf(h, "o=%s:%s:%d:%d|", o.ID, o.Action, o.SourceStartMs, o.SourceEndMs)
	}
	for _, d := range decisions {
		_, _ = fmt.Fprintf(h, "dec=%s:%s|", d.DetectionID, d.Action)
	}
	for _, s := range segments {
		_, _ = fmt.Fprintf(h, "s=%d:%d:%d|", s.SegmentIndex, s.SourceStartMs, s.SourceEndMs)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

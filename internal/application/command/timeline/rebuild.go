package timeline

import (
	"context"
	"errors"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	domaintimeline "go-api/internal/domain/timeline"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type RebuildCommand struct {
	ProjectID uuid.UUID
	MediaID   uuid.UUID
}

type RebuildHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.ReadRepository
	timelines  domaintimeline.WriteRepository
	timelineR  domaintimeline.ReadRepository
}

func NewRebuildHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.ReadRepository,
	timelines domaintimeline.WriteRepository,
	timelineR domaintimeline.ReadRepository,
) *RebuildHandler {
	return &RebuildHandler{media: media, detections: detections, timelines: timelines, timelineR: timelineR}
}

func (h *RebuildHandler) Handle(ctx context.Context, cmd RebuildCommand) (*domaintimeline.Timeline, eng.Output, error) {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, eng.Output{}, err
	}
	if m == nil || m.ProjectID != cmd.ProjectID {
		return nil, eng.Output{}, errors.New("media not found")
	}
	out, err := buildForMedia(ctx, h.media, h.detections, m)
	if err != nil {
		return nil, eng.Output{}, err
	}

	head, err := h.timelineR.FindHeadByMedia(ctx, cmd.MediaID)
	if err != nil {
		return nil, eng.Output{}, err
	}
	if head != nil && head.Fingerprint == out.Fingerprint {
		return head, out, nil
	}

	segments := mapSegments(out)
	decisions := mapDecisions(out)

	if head == nil {
		t := domaintimeline.New(cmd.ProjectID, cmd.MediaID, 1, out.Fingerprint, out.DurationMs)
		err = h.timelines.WithTransaction(ctx, func(txCtx context.Context) error {
			return h.timelines.Save(txCtx, t, segments, decisions)
		})
		if err != nil {
			return nil, eng.Output{}, err
		}
		return t, out, nil
	}

	head.Fingerprint = out.Fingerprint
	head.DurationMs = out.DurationMs
	head.Status = domaintimeline.StatusDraft
	err = h.timelines.WithTransaction(ctx, func(txCtx context.Context) error {
		return h.timelines.ReplaceContents(txCtx, head, segments, decisions)
	})
	if err != nil {
		return nil, eng.Output{}, err
	}
	return head, out, nil
}

func mapSegments(out eng.Output) []domaintimeline.Segment {
	segments := make([]domaintimeline.Segment, 0, len(out.Segments))
	for _, s := range out.Segments {
		var reason *domaintimeline.CutReason
		if s.RemovedBeforeReason != nil {
			r := domaintimeline.CutReason(*s.RemovedBeforeReason)
			reason = &r
		}
		segments = append(segments, domaintimeline.Segment{
			ID:                  uuid.New(),
			SegmentIndex:        s.SegmentIndex,
			MediaFileID:         s.MediaFileID,
			SourceStartMs:       s.SourceStartMs,
			SourceEndMs:         s.SourceEndMs,
			OutputStartMs:       s.OutputStartMs,
			OutputEndMs:         s.OutputEndMs,
			RemovedBeforeReason: reason,
		})
	}
	return segments
}

func mapDecisions(out eng.Output) []domaintimeline.Decision {
	decisions := make([]domaintimeline.Decision, 0, len(out.Decisions))
	for _, d := range out.Decisions {
		decisions = append(decisions, domaintimeline.Decision{
			DetectionID: d.DetectionID,
			Action:      domaintimeline.DecisionAction(d.Action),
			Origin:      domaintimeline.DecisionOrigin(d.Origin),
			Reason:      d.Reason,
		})
	}
	return decisions
}

package presenter

import (
	"time"

	querytimeline "go-api/internal/application/query/timeline"
	domaintimeline "go-api/internal/domain/timeline"
	eng "go-api/internal/timeline"
)

type TimelineSegmentResponse struct {
	ID                  string  `json:"id,omitempty"`
	SegmentIndex        int     `json:"segmentIndex"`
	MediaFileID         string  `json:"mediaFileId"`
	SourceStartMs       int64   `json:"sourceStartMs"`
	SourceEndMs         int64   `json:"sourceEndMs"`
	OutputStartMs       int64   `json:"outputStartMs"`
	OutputEndMs         int64   `json:"outputEndMs"`
	RemovedBeforeReason *string `json:"removedBeforeReason,omitempty"`
}

type TimelineDecisionResponse struct {
	DetectionID string  `json:"detectionId"`
	Action      string  `json:"action"`
	Origin      string  `json:"origin"`
	Reason      *string `json:"reason,omitempty"`
}

type TimelineResponse struct {
	ID          string                     `json:"id,omitempty"`
	ProjectID   string                     `json:"projectId"`
	Version     int                        `json:"version,omitempty"`
	Status      string                     `json:"status,omitempty"`
	Fingerprint string                     `json:"fingerprint"`
	DurationMs  int64                      `json:"durationMs"`
	ValidatedAt *time.Time                 `json:"validatedAt,omitempty"`
	CreatedAt   *time.Time                 `json:"createdAt,omitempty"`
	Segments    []TimelineSegmentResponse  `json:"segments"`
	Decisions   []TimelineDecisionResponse `json:"decisions"`
}

func NewTimelineResponseFromView(v *querytimeline.TimelineView) TimelineResponse {
	t := v.Timeline
	segments := make([]TimelineSegmentResponse, 0, len(v.Segments))
	for _, s := range v.Segments {
		var reason *string
		if s.RemovedBeforeReason != nil {
			r := string(*s.RemovedBeforeReason)
			reason = &r
		}
		segments = append(segments, TimelineSegmentResponse{
			ID:                  s.ID.String(),
			SegmentIndex:        s.SegmentIndex,
			MediaFileID:         s.MediaFileID.String(),
			SourceStartMs:       s.SourceStartMs,
			SourceEndMs:         s.SourceEndMs,
			OutputStartMs:       s.OutputStartMs,
			OutputEndMs:         s.OutputEndMs,
			RemovedBeforeReason: reason,
		})
	}
	decisions := make([]TimelineDecisionResponse, 0, len(v.Decisions))
	for _, d := range v.Decisions {
		decisions = append(decisions, TimelineDecisionResponse{
			DetectionID: d.DetectionID.String(),
			Action:      string(d.Action),
			Origin:      string(d.Origin),
			Reason:      d.Reason,
		})
	}
	created := t.CreatedAt
	return TimelineResponse{
		ID:          t.ID.String(),
		ProjectID:   t.ProjectID.String(),
		Version:     t.Version,
		Status:      string(t.Status),
		Fingerprint: t.Fingerprint,
		DurationMs:  t.DurationMs,
		ValidatedAt: t.ValidatedAt,
		CreatedAt:   &created,
		Segments:    segments,
		Decisions:   decisions,
	}
}

func NewTimelinePreviewResponse(projectID string, out eng.Output) TimelineResponse {
	segments := make([]TimelineSegmentResponse, 0, len(out.Segments))
	for _, s := range out.Segments {
		var reason *string
		if s.RemovedBeforeReason != nil {
			r := string(*s.RemovedBeforeReason)
			reason = &r
		}
		segments = append(segments, TimelineSegmentResponse{
			SegmentIndex:        s.SegmentIndex,
			MediaFileID:         s.MediaFileID.String(),
			SourceStartMs:       s.SourceStartMs,
			SourceEndMs:         s.SourceEndMs,
			OutputStartMs:       s.OutputStartMs,
			OutputEndMs:         s.OutputEndMs,
			RemovedBeforeReason: reason,
		})
	}
	decisions := make([]TimelineDecisionResponse, 0, len(out.Decisions))
	for _, d := range out.Decisions {
		decisions = append(decisions, TimelineDecisionResponse{
			DetectionID: d.DetectionID.String(),
			Action:      string(d.Action),
			Origin:      d.Origin,
			Reason:      d.Reason,
		})
	}
	return TimelineResponse{
		ProjectID:   projectID,
		Fingerprint: out.Fingerprint,
		DurationMs:  out.DurationMs,
		Segments:    segments,
		Decisions:   decisions,
	}
}

func NewTimelineResponseFromEntity(t *domaintimeline.Timeline, out eng.Output) TimelineResponse {
	resp := NewTimelinePreviewResponse(t.ProjectID.String(), out)
	resp.ID = t.ID.String()
	resp.Version = t.Version
	resp.Status = string(t.Status)
	resp.ValidatedAt = t.ValidatedAt
	created := t.CreatedAt
	resp.CreatedAt = &created
	return resp
}

type TimelineVersionResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"projectId"`
	Version     int        `json:"version"`
	Status      string     `json:"status"`
	Fingerprint string     `json:"fingerprint"`
	DurationMs  int64      `json:"durationMs"`
	ValidatedAt *time.Time `json:"validatedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func NewTimelineVersionListResponse(items []domaintimeline.Timeline) []TimelineVersionResponse {
	out := make([]TimelineVersionResponse, 0, len(items))
	for _, t := range items {
		out = append(out, TimelineVersionResponse{
			ID:          t.ID.String(),
			ProjectID:   t.ProjectID.String(),
			Version:     t.Version,
			Status:      string(t.Status),
			Fingerprint: t.Fingerprint,
			DurationMs:  t.DurationMs,
			ValidatedAt: t.ValidatedAt,
			CreatedAt:   t.CreatedAt,
		})
	}
	return out
}

package timeline

import (
	"context"
	"errors"

	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
)

type GetTimelineQuery struct {
	ProjectID uuid.UUID
}

type TimelineView struct {
	Timeline  *domaintimeline.Timeline
	Segments  []domaintimeline.Segment
	Decisions []domaintimeline.Decision
}

type GetTimelineHandler struct {
	repo domaintimeline.ReadRepository
}

func NewGetTimelineHandler(repo domaintimeline.ReadRepository) *GetTimelineHandler {
	return &GetTimelineHandler{repo: repo}
}

func (h *GetTimelineHandler) Handle(ctx context.Context, q GetTimelineQuery) (*TimelineView, error) {
	t, err := h.repo.FindHeadByProject(ctx, q.ProjectID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("timeline not found")
	}
	segments, err := h.repo.ListSegments(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	decisions, err := h.repo.ListDecisions(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	return &TimelineView{Timeline: t, Segments: segments, Decisions: decisions}, nil
}

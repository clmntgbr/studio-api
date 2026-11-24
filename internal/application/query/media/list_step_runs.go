package media

import (
	"context"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type ListStepRunsQuery struct {
	MediaID uuid.UUID
}

type ListStepRunsResult struct {
	Steps []domainmedia.StepView
	Runs  []domainmedia.StepRunView
}

type ListStepRunsHandler struct {
	repo domainmedia.ReadRepository
}

func NewListStepRunsHandler(repo domainmedia.ReadRepository) *ListStepRunsHandler {
	return &ListStepRunsHandler{repo: repo}
}

func (h *ListStepRunsHandler) Handle(ctx context.Context, q ListStepRunsQuery) (*ListStepRunsResult, error) {
	steps, err := h.repo.ListSteps(ctx, q.MediaID)
	if err != nil {
		return nil, err
	}
	runs, err := h.repo.ListStepRuns(ctx, q.MediaID)
	if err != nil {
		return nil, err
	}
	return &ListStepRunsResult{Steps: steps, Runs: runs}, nil
}

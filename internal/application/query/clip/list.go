package clip

import (
	"context"

	domainclip "go-api/internal/domain/clip"

	"github.com/google/uuid"
)

type ListQuery struct {
	ProjectID uuid.UUID
}

type ListHandler struct {
	repo domainclip.ReadRepository
}

func NewListHandler(repo domainclip.ReadRepository) *ListHandler {
	return &ListHandler{repo: repo}
}

func (h *ListHandler) Handle(ctx context.Context, q ListQuery) ([]domainclip.Clip, error) {
	return h.repo.ListByProject(ctx, q.ProjectID)
}

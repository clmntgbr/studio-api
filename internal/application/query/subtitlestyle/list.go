package subtitlestyle

import (
	"context"

	domainstyle "go-api/internal/domain/subtitlestyle"

	"github.com/google/uuid"
)

type ListQuery struct {
	ProjectID *uuid.UUID
}

type ListHandler struct {
	repo domainstyle.ReadRepository
}

func NewListHandler(repo domainstyle.ReadRepository) *ListHandler {
	return &ListHandler{repo: repo}
}

func (h *ListHandler) Handle(ctx context.Context, q ListQuery) ([]domainstyle.Style, error) {
	return h.repo.List(ctx, q.ProjectID)
}

package render

import (
	"context"

	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type ListQuery struct {
	MediaID uuid.UUID
}

type ListHandler struct {
	repo domainrender.ReadRepository
}

func NewListHandler(repo domainrender.ReadRepository) *ListHandler {
	return &ListHandler{repo: repo}
}

func (h *ListHandler) Handle(ctx context.Context, q ListQuery) ([]domainrender.Render, error) {
	return h.repo.ListByMedia(ctx, q.MediaID)
}

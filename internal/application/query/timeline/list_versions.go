package timeline

import (
	"context"

	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
)

type ListVersionsQuery struct {
	MediaID uuid.UUID
}

type ListVersionsHandler struct {
	repo domaintimeline.ReadRepository
}

func NewListVersionsHandler(repo domaintimeline.ReadRepository) *ListVersionsHandler {
	return &ListVersionsHandler{repo: repo}
}

func (h *ListVersionsHandler) Handle(ctx context.Context, q ListVersionsQuery) ([]domaintimeline.Timeline, error) {
	return h.repo.ListByMedia(ctx, q.MediaID)
}

package media

import (
	"context"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/paginate"

	"github.com/google/uuid"
)

type ListMediaQuery struct {
	ProjectID uuid.UUID
	Query     paginate.PaginateQuery
}

type ListMediaHandler struct {
	repo domainmedia.ReadRepository
}

func NewListMediaHandler(repo domainmedia.ReadRepository) *ListMediaHandler {
	return &ListMediaHandler{repo: repo}
}

func (h *ListMediaHandler) Handle(ctx context.Context, q ListMediaQuery) ([]domainmedia.FileView, int64, error) {
	return h.repo.ListByProject(ctx, q.ProjectID, q.Query)
}

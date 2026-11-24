package project

import (
	"context"

	"go-api/internal/domain/paginate"
	domainproject "go-api/internal/domain/project"
)

type ListProjectsQuery struct {
	Query paginate.PaginateQuery
}

type ListProjectsHandler struct {
	repo domainproject.ReadRepository
}

func NewListProjectsHandler(repo domainproject.ReadRepository) *ListProjectsHandler {
	return &ListProjectsHandler{repo: repo}
}

func (h *ListProjectsHandler) Handle(ctx context.Context, q ListProjectsQuery) ([]domainproject.View, int64, error) {
	return h.repo.List(ctx, q.Query)
}

package subtitlestyle

import (
	"context"

	domainstyle "go-api/internal/domain/subtitlestyle"
)

type ListQuery struct{}

type ListHandler struct {
	repo domainstyle.ReadRepository
}

func NewListHandler(repo domainstyle.ReadRepository) *ListHandler {
	return &ListHandler{repo: repo}
}

func (h *ListHandler) Handle(ctx context.Context, _ ListQuery) ([]domainstyle.Style, error) {
	return h.repo.List(ctx)
}
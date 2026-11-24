package outputpreset

import (
	"context"

	domainpreset "go-api/internal/domain/outputpreset"
)

type ListHandler struct {
	repo domainpreset.ReadRepository
}

func NewListHandler(repo domainpreset.ReadRepository) *ListHandler {
	return &ListHandler{repo: repo}
}

func (h *ListHandler) Handle(ctx context.Context) ([]domainpreset.Preset, error) {
	return h.repo.List(ctx)
}

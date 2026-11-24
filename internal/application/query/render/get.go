package render

import (
	"context"
	"errors"

	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type GetQuery struct {
	ID uuid.UUID
}

type GetHandler struct {
	repo domainrender.ReadRepository
}

func NewGetHandler(repo domainrender.ReadRepository) *GetHandler {
	return &GetHandler{repo: repo}
}

func (h *GetHandler) Handle(ctx context.Context, q GetQuery) (*domainrender.Render, error) {
	r, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("render not found")
	}
	return r, nil
}

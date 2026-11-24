package clip

import (
	"context"
	"errors"

	domainclip "go-api/internal/domain/clip"

	"github.com/google/uuid"
)

type GetQuery struct {
	ID uuid.UUID
}

type GetHandler struct {
	repo domainclip.ReadRepository
}

func NewGetHandler(repo domainclip.ReadRepository) *GetHandler {
	return &GetHandler{repo: repo}
}

func (h *GetHandler) Handle(ctx context.Context, q GetQuery) (*domainclip.Clip, error) {
	c, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("clip not found")
	}
	return c, nil
}

package subtitlestyle

import (
	"context"
	"errors"

	domainstyle "go-api/internal/domain/subtitlestyle"

	"github.com/google/uuid"
)

type GetQuery struct {
	ID uuid.UUID
}

type GetHandler struct {
	repo domainstyle.ReadRepository
}

func NewGetHandler(repo domainstyle.ReadRepository) *GetHandler {
	return &GetHandler{repo: repo}
}

func (h *GetHandler) Handle(ctx context.Context, q GetQuery) (*domainstyle.Style, error) {
	s, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errors.New("subtitle style not found")
	}
	return s, nil
}

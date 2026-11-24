package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type GetMediaQuery struct {
	ID uuid.UUID
}

type GetMediaHandler struct {
	repo domainmedia.ReadRepository
}

func NewGetMediaHandler(repo domainmedia.ReadRepository) *GetMediaHandler {
	return &GetMediaHandler{repo: repo}
}

func (h *GetMediaHandler) Handle(ctx context.Context, q GetMediaQuery) (*domainmedia.FileView, error) {
	view, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, errors.New("media not found")
	}
	return view, nil
}

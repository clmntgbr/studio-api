package project

import (
	"context"
	"errors"

	domainproject "go-api/internal/domain/project"

	"github.com/google/uuid"
)

type GetProjectQuery struct {
	ID uuid.UUID
}

type GetProjectHandler struct {
	repo domainproject.ReadRepository
}

func NewGetProjectHandler(repo domainproject.ReadRepository) *GetProjectHandler {
	return &GetProjectHandler{repo: repo}
}

func (h *GetProjectHandler) Handle(ctx context.Context, q GetProjectQuery) (*domainproject.View, error) {
	view, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, errors.New("project not found")
	}
	return view, nil
}

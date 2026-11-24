package project

import (
	"context"
	"errors"

	domainproject "go-api/internal/domain/project"

	"github.com/google/uuid"
)

type UpdateProjectCommand struct {
	ID   uuid.UUID
	Name string
}

type UpdateProjectHandler struct {
	repo domainproject.WriteRepository
}

func NewUpdateProjectHandler(repo domainproject.WriteRepository) *UpdateProjectHandler {
	return &UpdateProjectHandler{repo: repo}
}

func (h *UpdateProjectHandler) Handle(ctx context.Context, cmd UpdateProjectCommand) (*domainproject.Project, error) {
	p, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("project not found")
	}
	if cmd.Name != "" {
		p.Rename(cmd.Name)
	}
	if err := h.repo.Update(ctx, p); err != nil {
		return nil, errors.New("failed to update project")
	}
	return p, nil
}

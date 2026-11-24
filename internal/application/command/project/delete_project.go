package project

import (
	"context"
	"errors"

	domainproject "go-api/internal/domain/project"

	"github.com/google/uuid"
)

type DeleteProjectCommand struct {
	ID uuid.UUID
}

type DeleteProjectHandler struct {
	repo domainproject.WriteRepository
}

func NewDeleteProjectHandler(repo domainproject.WriteRepository) *DeleteProjectHandler {
	return &DeleteProjectHandler{repo: repo}
}

func (h *DeleteProjectHandler) Handle(ctx context.Context, cmd DeleteProjectCommand) error {
	p, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("project not found")
	}
	if err := h.repo.Delete(ctx, cmd.ID); err != nil {
		return errors.New("failed to delete project")
	}
	return nil
}

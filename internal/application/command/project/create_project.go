package project

import (
	"context"
	"errors"

	"go-api/internal/domain/port"
	domainproject "go-api/internal/domain/project"
)

type CreateProjectCommand struct {
	Name string
}

type CreateProjectHandler struct {
	repo   domainproject.WriteRepository
	outbox port.OutboxRepository
}

func NewCreateProjectHandler(repo domainproject.WriteRepository, outbox port.OutboxRepository) *CreateProjectHandler {
	return &CreateProjectHandler{repo: repo, outbox: outbox}
}

func (h *CreateProjectHandler) Handle(ctx context.Context, cmd CreateProjectCommand) (*domainproject.Project, error) {
	if cmd.Name == "" {
		return nil, errors.New("name is required")
	}
	p := domainproject.NewProject(cmd.Name)
	err := h.repo.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.repo.Save(txCtx, p); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, p.PullEvents())
	})
	if err != nil {
		return nil, errors.New("failed to create project")
	}
	return p, nil
}

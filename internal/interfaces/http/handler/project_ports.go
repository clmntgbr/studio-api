package handler

import (
	"context"

	cmdproject "go-api/internal/application/command/project"
	queryproject "go-api/internal/application/query/project"
	domainproject "go-api/internal/domain/project"
)

type projectCreateHandler interface {
	Handle(ctx context.Context, cmd cmdproject.CreateProjectCommand) (*domainproject.Project, error)
}

type projectUpdateHandler interface {
	Handle(ctx context.Context, cmd cmdproject.UpdateProjectCommand) (*domainproject.Project, error)
}

type projectDeleteHandler interface {
	Handle(ctx context.Context, cmd cmdproject.DeleteProjectCommand) error
}

type projectGetHandler interface {
	Handle(ctx context.Context, q queryproject.GetProjectQuery) (*domainproject.View, error)
}

type projectListHandler interface {
	Handle(ctx context.Context, q queryproject.ListProjectsQuery) ([]domainproject.View, int64, error)
}

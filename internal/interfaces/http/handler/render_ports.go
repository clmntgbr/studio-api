package handler

import (
	"context"

	cmdrender "go-api/internal/application/command/render"
	queryrender "go-api/internal/application/query/render"
	domainrender "go-api/internal/domain/render"
)

type renderCreateHandler interface {
	Handle(ctx context.Context, cmd cmdrender.CreateCommand) (*domainrender.Render, error)
}

type renderCancelHandler interface {
	Handle(ctx context.Context, cmd cmdrender.CancelCommand) (*domainrender.Render, error)
}

type renderGetHandler interface {
	Handle(ctx context.Context, q queryrender.GetQuery) (*domainrender.Render, error)
}

type renderListHandler interface {
	Handle(ctx context.Context, q queryrender.ListQuery) ([]domainrender.Render, error)
}

type renderDownloadHandler interface {
	Handle(ctx context.Context, q queryrender.DownloadQuery) (*queryrender.DownloadResult, error)
}

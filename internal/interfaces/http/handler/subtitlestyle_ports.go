package handler

import (
	"context"

	cmdstyle "go-api/internal/application/command/subtitlestyle"
	querystyle "go-api/internal/application/query/subtitlestyle"
	domainstyle "go-api/internal/domain/subtitlestyle"
)

type subtitleStyleCreateHandler interface {
	Handle(ctx context.Context, cmd cmdstyle.CreateCommand) (*domainstyle.Style, error)
}

type subtitleStyleUpdateHandler interface {
	Handle(ctx context.Context, cmd cmdstyle.UpdateCommand) (*domainstyle.Style, error)
}

type subtitleStyleDeleteHandler interface {
	Handle(ctx context.Context, cmd cmdstyle.DeleteCommand) error
}

type subtitleStyleGetHandler interface {
	Handle(ctx context.Context, q querystyle.GetQuery) (*domainstyle.Style, error)
}

type subtitleStyleListHandler interface {
	Handle(ctx context.Context, q querystyle.ListQuery) ([]domainstyle.Style, error)
}

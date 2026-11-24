package handler

import (
	"context"

	cmdclip "go-api/internal/application/command/clip"
	cmdrender "go-api/internal/application/command/render"
	queryclip "go-api/internal/application/query/clip"
	queryllm "go-api/internal/application/query/llmartifact"
	domainclip "go-api/internal/domain/clip"
	domainllm "go-api/internal/domain/llmartifact"
	domainrender "go-api/internal/domain/render"
)

type clipListHandler interface {
	Handle(ctx context.Context, q queryclip.ListQuery) ([]domainclip.Clip, error)
}

type clipCreateHandler interface {
	Handle(ctx context.Context, cmd cmdclip.CreateCommand) (*domainclip.Clip, error)
}

type clipUpdateHandler interface {
	Handle(ctx context.Context, cmd cmdclip.UpdateCommand) (*domainclip.Clip, error)
}

type clipAnalyzeHandler interface {
	Handle(ctx context.Context, cmd cmdclip.AnalyzeCommand) error
}

type llmArtifactListHandler interface {
	Handle(ctx context.Context, q queryllm.ListQuery) ([]domainllm.Artifact, error)
}

type clipGetHandler interface {
	Handle(ctx context.Context, q queryclip.GetQuery) (*domainclip.Clip, error)
}

type clipRenderHandler interface {
	Handle(ctx context.Context, cmd cmdrender.CreateCommand) (*domainrender.Render, error)
}

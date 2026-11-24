package handler

import (
	"context"

	cmdmedia "go-api/internal/application/command/media"
)

type minioMarkSourceReadyHandler interface {
	Handle(ctx context.Context, cmd cmdmedia.MarkSourceReadyCommand) error
}

package handler

import (
	"context"

	cmddetection "go-api/internal/application/command/detection"
	domaindetection "go-api/internal/domain/detection"
)

type overrideCreateHandler interface {
	Handle(ctx context.Context, cmd cmddetection.CreateOverrideCommand) (*domaindetection.Override, error)
}

type overrideDeleteHandler interface {
	Handle(ctx context.Context, cmd cmddetection.DeleteOverrideCommand) error
}

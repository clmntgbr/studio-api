package handler

import (
	"context"

	cmdmedia "go-api/internal/application/command/media"
	querymedia "go-api/internal/application/query/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
)

type mediaInitiateUploadHandler interface {
	Handle(ctx context.Context, cmd cmdmedia.InitiateUploadCommand) (*cmdmedia.InitiateUploadResult, error)
}

type mediaPresignPartsHandler interface {
	Handle(ctx context.Context, cmd cmdmedia.PresignPartsCommand) ([]port.UploadPartURL, error)
}

type mediaCompleteUploadHandler interface {
	Handle(ctx context.Context, cmd cmdmedia.CompleteUploadCommand) (*domainmedia.MediaFile, error)
}

type mediaFromURLHandler interface {
	Handle(ctx context.Context, cmd cmdmedia.FromURLCommand) (*domainmedia.MediaFile, error)
}

type mediaDeleteHandler interface {
	Handle(ctx context.Context, cmd cmdmedia.DeleteMediaCommand) error
}

type mediaGetHandler interface {
	Handle(ctx context.Context, q querymedia.GetMediaQuery) (*domainmedia.FileView, error)
}

type mediaListHandler interface {
	Handle(ctx context.Context, q querymedia.ListMediaQuery) ([]domainmedia.FileView, int64, error)
}

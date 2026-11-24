package handler

import (
	"context"

	cmdtimeline "go-api/internal/application/command/timeline"
	querytimeline "go-api/internal/application/query/timeline"
	domaintimeline "go-api/internal/domain/timeline"
)

type timelinePreviewHandler interface {
	Handle(ctx context.Context, cmd cmdtimeline.PreviewCommand) (*cmdtimeline.PreviewResult, error)
}

type timelineValidateHandler interface {
	Handle(ctx context.Context, cmd cmdtimeline.ValidateCommand) (*domaintimeline.Timeline, error)
}

type timelineGetHandler interface {
	Handle(ctx context.Context, q querytimeline.GetTimelineQuery) (*querytimeline.TimelineView, error)
}

type timelineListVersionsHandler interface {
	Handle(ctx context.Context, q querytimeline.ListVersionsQuery) ([]domaintimeline.Timeline, error)
}

type timelineGetSubtitlesHandler interface {
	Handle(ctx context.Context, q querytimeline.GetSubtitlesQuery) (string, string, error)
}

type timelineGetExportHandler interface {
	Handle(ctx context.Context, q querytimeline.GetExportQuery) (body string, contentType string, filename string, err error)
}

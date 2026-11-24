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

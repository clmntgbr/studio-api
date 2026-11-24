package timeline

import (
	"context"
	"errors"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type PreviewCommand struct {
	ProjectID uuid.UUID
	MediaID   uuid.UUID // optional: first media of project if empty
}

type PreviewResult struct {
	Output eng.Output
}

type PreviewHandler struct {
	media      domainmedia.WriteRepository
	mediaRead  domainmedia.ReadRepository
	detections domaindetection.ReadRepository
}

func NewPreviewHandler(
	media domainmedia.WriteRepository,
	mediaRead domainmedia.ReadRepository,
	detections domaindetection.ReadRepository,
) *PreviewHandler {
	return &PreviewHandler{media: media, mediaRead: mediaRead, detections: detections}
}

func (h *PreviewHandler) Handle(ctx context.Context, cmd PreviewCommand) (*PreviewResult, error) {
	mediaID := cmd.MediaID
	if mediaID == uuid.Nil {
		views, _, err := h.mediaRead.ListByProject(ctx, cmd.ProjectID, paginateAll())
		if err != nil {
			return nil, err
		}
		if len(views) == 0 {
			return nil, errors.New("no media in project")
		}
		mediaID = views[0].ID
	}
	m, err := h.media.GetByID(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.ProjectID != cmd.ProjectID {
		return nil, errors.New("media not found")
	}
	out, err := buildForMedia(ctx, h.media, h.detections, m)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{Output: out}, nil
}

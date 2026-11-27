package timeline

import (
	"context"
	"errors"

	mediacmd "go-api/internal/application/command/media"
	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	eng "go-api/internal/timeline"

	"github.com/google/uuid"
)

type PreviewCommand struct {
	MediaID uuid.UUID
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
	if cmd.MediaID == uuid.Nil {
		return nil, errors.New("mediaId is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("media not found")
	}
	if !m.Editable() {
		return nil, mediacmd.ErrMediaNotEditable
	}
	out, err := buildForMedia(ctx, h.media, h.detections, m)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{Output: out}, nil
}

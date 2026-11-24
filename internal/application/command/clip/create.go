package clip

import (
	"context"
	"errors"

	domainclip "go-api/internal/domain/clip"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type CreateCommand struct {
	ProjectID     uuid.UUID
	MediaFileID   uuid.UUID
	SourceStartMs int64
	SourceEndMs   int64
	Title         *string
}

type CreateHandler struct {
	clips domainclip.WriteRepository
	media domainmedia.WriteRepository
}

func NewCreateHandler(clips domainclip.WriteRepository, media domainmedia.WriteRepository) *CreateHandler {
	return &CreateHandler{clips: clips, media: media}
}

func (h *CreateHandler) Handle(ctx context.Context, cmd CreateCommand) (*domainclip.Clip, error) {
	if cmd.SourceEndMs <= cmd.SourceStartMs {
		return nil, errors.New("sourceEndMs must be greater than sourceStartMs")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaFileID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.ProjectID != cmd.ProjectID {
		return nil, errors.New("media not found")
	}
	c := domainclip.New(cmd.ProjectID, cmd.MediaFileID, cmd.SourceStartMs, cmd.SourceEndMs, "manual")
	c.Title = cmd.Title
	c.Selected = true
	if err := h.clips.Save(ctx, c); err != nil {
		return nil, errors.New("failed to create clip")
	}
	return c, nil
}

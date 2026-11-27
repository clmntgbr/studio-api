package clip

import (
	"context"
	"errors"

	mediacmd "go-api/internal/application/command/media"
	domainclip "go-api/internal/domain/clip"
	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type UpdateCommand struct {
	ClipID        uuid.UUID
	Selected      *bool
	SourceStartMs *int64
	SourceEndMs   *int64
	Title         *string
	Description   *string
	Hashtags      *[]string
}

type UpdateHandler struct {
	clips domainclip.WriteRepository
	read  domainclip.ReadRepository
	media domainmedia.WriteRepository
}

func NewUpdateHandler(
	clips domainclip.WriteRepository,
	read domainclip.ReadRepository,
	media domainmedia.WriteRepository,
) *UpdateHandler {
	return &UpdateHandler{clips: clips, read: read, media: media}
}

func (h *UpdateHandler) Handle(ctx context.Context, cmd UpdateCommand) (*domainclip.Clip, error) {
	c, err := h.read.FindByID(ctx, cmd.ClipID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("clip not found")
	}
	if h.media != nil {
		m, err := h.media.GetByID(ctx, c.MediaFileID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, errors.New("media not found")
		}
		if !m.Editable() {
			return nil, mediacmd.ErrMediaNotEditable
		}
	}
	if cmd.Selected != nil {
		c.Selected = *cmd.Selected
	}
	if cmd.SourceStartMs != nil {
		c.SourceStartMs = *cmd.SourceStartMs
	}
	if cmd.SourceEndMs != nil {
		c.SourceEndMs = *cmd.SourceEndMs
	}
	if c.SourceEndMs <= c.SourceStartMs {
		return nil, errors.New("sourceEndMs must be greater than sourceStartMs")
	}
	if cmd.Title != nil {
		c.Title = cmd.Title
	}
	if cmd.Description != nil {
		c.Description = cmd.Description
	}
	if cmd.Hashtags != nil {
		c.Hashtags = *cmd.Hashtags
	}
	if err := h.clips.Update(ctx, c); err != nil {
		return nil, errors.New("failed to update clip")
	}
	return c, nil
}

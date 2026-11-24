package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type DeleteMediaCommand struct {
	ID uuid.UUID
}

type DeleteMediaHandler struct {
	media domainmedia.WriteRepository
}

func NewDeleteMediaHandler(media domainmedia.WriteRepository) *DeleteMediaHandler {
	return &DeleteMediaHandler{media: media}
}

func (h *DeleteMediaHandler) Handle(ctx context.Context, cmd DeleteMediaCommand) error {
	m, err := h.media.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}
	if err := h.media.Delete(ctx, cmd.ID); err != nil {
		return errors.New("failed to delete media")
	}
	return nil
}

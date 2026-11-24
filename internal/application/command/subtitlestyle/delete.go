package subtitlestyle

import (
	"context"
	"errors"

	domainstyle "go-api/internal/domain/subtitlestyle"

	"github.com/google/uuid"
)

type DeleteCommand struct {
	ID uuid.UUID
}

type DeleteHandler struct {
	repo domainstyle.WriteRepository
}

func NewDeleteHandler(repo domainstyle.WriteRepository) *DeleteHandler {
	return &DeleteHandler{repo: repo}
}

func (h *DeleteHandler) Handle(ctx context.Context, cmd DeleteCommand) error {
	s, err := h.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if s == nil {
		return errors.New("subtitle style not found")
	}
	if err := h.repo.Delete(ctx, cmd.ID); err != nil {
		return errors.New("failed to delete subtitle style")
	}
	return nil
}

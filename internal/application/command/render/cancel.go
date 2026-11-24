package render

import (
	"context"
	"errors"
	"time"

	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type CancelCommand struct {
	RenderID uuid.UUID
}

type CancelHandler struct {
	renders domainrender.WriteRepository
}

func NewCancelHandler(renders domainrender.WriteRepository) *CancelHandler {
	return &CancelHandler{renders: renders}
}

func (h *CancelHandler) Handle(ctx context.Context, cmd CancelCommand) (*domainrender.Render, error) {
	r, err := h.renders.GetByID(ctx, cmd.RenderID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("render not found")
	}
	switch r.Status {
	case domainrender.StatusCompleted, domainrender.StatusFailed, domainrender.StatusCancelled:
		return r, nil
	}
	now := time.Now().UTC()
	r.Status = domainrender.StatusCancelled
	r.CompletedAt = &now
	if err := h.renders.Update(ctx, r); err != nil {
		return nil, errors.New("failed to cancel render")
	}
	return r, nil
}

package render

import (
	"context"
	"errors"
	"time"

	mediacmd "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type CancelCommand struct {
	RenderID uuid.UUID
}

type CancelHandler struct {
	renders domainrender.WriteRepository
	media   domainmedia.WriteRepository
}

func NewCancelHandler(renders domainrender.WriteRepository, media domainmedia.WriteRepository) *CancelHandler {
	return &CancelHandler{renders: renders, media: media}
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
	if r.Kind == domainrender.KindMaster && h.media != nil && r.MediaFileID != uuid.Nil {
		_ = mediacmd.FailStep(ctx, h.media, r.MediaFileID, domainmedia.StepRender, now, "cancelled")
	}
	_ = mediacmd.ReturnMediaToReadyIfIdle(ctx, h.media, h.renders.HasInFlightByMedia, r.MediaFileID)
	return r, nil
}

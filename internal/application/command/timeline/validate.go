package timeline

import (
	"context"
	"errors"

	domaintimeline "go-api/internal/domain/timeline"

	"github.com/google/uuid"
)

type ValidateCommand struct {
	MediaID uuid.UUID
	Version int
}

type ValidateHandler struct {
	timelines domaintimeline.WriteRepository
	timelineR domaintimeline.ReadRepository
}

func NewValidateHandler(timelines domaintimeline.WriteRepository, timelineR domaintimeline.ReadRepository) *ValidateHandler {
	return &ValidateHandler{timelines: timelines, timelineR: timelineR}
}

func (h *ValidateHandler) Handle(ctx context.Context, cmd ValidateCommand) (*domaintimeline.Timeline, error) {
	head, err := h.timelineR.FindHeadByMedia(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if head == nil {
		return nil, errors.New("timeline not found")
	}
	if head.Version != cmd.Version {
		return nil, ErrStaleVersion
	}
	if head.Status == domaintimeline.StatusValidated {
		return head, nil
	}
	head.Validate()
	if err := h.timelines.Update(ctx, head); err != nil {
		return nil, errors.New("failed to validate timeline")
	}
	return head, nil
}

var ErrStaleVersion = errors.New("timeline version is stale")

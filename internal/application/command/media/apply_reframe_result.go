package media

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type ApplyReframeResultCommand struct {
	MediaID     uuid.UUID
	CropPathKey string
	SampleCount int
	Detector    string
}

type ApplyReframeResultHandler struct {
	media domainmedia.WriteRepository
}

func NewApplyReframeResultHandler(media domainmedia.WriteRepository) *ApplyReframeResultHandler {
	return &ApplyReframeResultHandler{media: media}
}

func (h *ApplyReframeResultHandler) Handle(ctx context.Context, cmd ApplyReframeResultCommand) error {
	if cmd.CropPathKey == "" {
		return errors.New("cropPathKey is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepReframe)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	now := time.Now().UTC()
	meta, _ := json.Marshal(map[string]any{
		"sampleCount": cmd.SampleCount,
		"detector":    cmd.Detector,
	})

	return h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.SaveAsset(txCtx, domainmedia.NewAsset(m.ID, domainmedia.AssetKindCropPath, cmd.CropPathKey, nil, meta)); err != nil {
			return err
		}
		return CompleteStep(txCtx, h.media, m.ID, domainmedia.StepReframe, now)
	})
}

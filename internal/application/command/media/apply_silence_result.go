package media

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	"go-api/internal/pipeline"

	"github.com/google/uuid"
)

type ApplySilenceResultCommand struct {
	MediaID               uuid.UUID
	DetectionsKey         string
	NoiseFloorDb          float64
	CalculatedThresholdDb float64
	Intervals             []pipeline.SilenceInterval
}

type ApplySilenceResultHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	outbox     port.OutboxRepository
}

func NewApplySilenceResultHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	outbox port.OutboxRepository,
) *ApplySilenceResultHandler {
	return &ApplySilenceResultHandler{media: media, detections: detections, outbox: outbox}
}

func (h *ApplySilenceResultHandler) Handle(ctx context.Context, cmd ApplySilenceResultCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepSilence)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	items := make([]*domaindetection.Detection, 0, len(cmd.Intervals))
	meta, _ := json.Marshal(map[string]any{
		"noiseFloorDb":          cmd.NoiseFloorDb,
		"calculatedThresholdDb": cmd.CalculatedThresholdDb,
		"detectionsKey":         cmd.DetectionsKey,
	})
	for _, iv := range cmd.Intervals {
		if iv.EndMs <= iv.StartMs {
			continue
		}
		d := domaindetection.New(m.ID, domaindetection.KindSilence, iv.StartMs, iv.EndMs, "ffmpeg.silencedetect", meta)
		items = append(items, d)
	}

	m.RecordDetectionsUpdated(string(domaindetection.KindSilence))
	now := time.Now().UTC()

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.detections.DeleteByMediaAndKind(txCtx, m.ID, domaindetection.KindSilence); err != nil {
			return err
		}
		if err := h.detections.SaveMany(txCtx, items); err != nil {
			return err
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := completeStep(txCtx, h.media, m.ID, domainmedia.StepSilence, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return errors.New("failed to apply silence result")
	}
	return nil
}

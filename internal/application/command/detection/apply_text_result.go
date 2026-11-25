package detection

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domaindetection "go-api/internal/domain/detection"
	mediacmd "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type TextInterval struct {
	Kind          string
	SourceStartMs int64
	SourceEndMs   int64
	FirstWord     *int
	LastWord      *int
	Confidence    *float64
}

type ApplyTextResultCommand struct {
	MediaID       uuid.UUID
	Attempt       int
	DetectionsKey string
	Detector      string
	Intervals     []TextInterval
}

type ApplyTextResultHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	outbox     port.OutboxRepository
}

func NewApplyTextResultHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	outbox port.OutboxRepository,
) *ApplyTextResultHandler {
	return &ApplyTextResultHandler{media: media, detections: detections, outbox: outbox}
}

func (h *ApplyTextResultHandler) Handle(ctx context.Context, cmd ApplyTextResultCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepText)
	if err != nil {
		return err
	}
	if step != nil && cmd.Attempt > 0 && cmd.Attempt != mediacmd.EffectiveStepAttempt(step.Attempt) {
		return nil // stale job for an older attempt
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted &&
		(cmd.Attempt == 0 || cmd.Attempt == mediacmd.EffectiveStepAttempt(step.Attempt)) {
		return nil
	}
	if step != nil && (step.Status == domainmedia.StepStatusCancelled ||
		step.Status == domainmedia.StepStatusSkipped) {
		return nil
	}

	detector := cmd.Detector
	if detector == "" {
		detector = "text.rules"
	}
	now := time.Now().UTC()
	textKinds := []domaindetection.Kind{
		domaindetection.KindFiller,
		domaindetection.KindRepetition,
		domaindetection.KindFalseStart,
	}
	items := make([]*domaindetection.Detection, 0, len(cmd.Intervals))
	metaBase, _ := json.Marshal(map[string]any{"detectionsKey": cmd.DetectionsKey})

	for _, iv := range cmd.Intervals {
		kind := domaindetection.Kind(iv.Kind)
		switch kind {
		case domaindetection.KindFiller, domaindetection.KindRepetition, domaindetection.KindFalseStart:
		default:
			continue
		}
		if iv.SourceEndMs <= iv.SourceStartMs {
			continue
		}
		d := domaindetection.New(m.ID, kind, iv.SourceStartMs, iv.SourceEndMs, detector, metaBase)
		d.FirstWordIndex = iv.FirstWord
		d.LastWordIndex = iv.LastWord
		d.Confidence = iv.Confidence
		items = append(items, d)
	}

	m.RecordDetectionsUpdated("text")

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, kind := range textKinds {
			if err := h.detections.DeleteByMediaAndKind(txCtx, m.ID, kind); err != nil {
				return err
			}
		}
		if err := h.detections.SaveMany(txCtx, items); err != nil {
			return err
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := mediacmd.CompleteStep(txCtx, h.media, m.ID, domainmedia.StepText, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return errors.New("failed to apply text result")
	}
	return nil
}

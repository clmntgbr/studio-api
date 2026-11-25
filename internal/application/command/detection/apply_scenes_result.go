package detection

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	mediacmd "go-api/internal/application/command/media"
	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ScenesAdCandidate struct {
	SourceStartMs int64
	SourceEndMs   int64
	Confidence    *float64
	Reason        string
	CutCount      int
}

type ApplyScenesResultCommand struct {
	MediaID         uuid.UUID
	ScenesKey       string
	AdCandidatesKey string
	Detector        string
	SceneCount      int
	Attempt         int
	AdCandidates    []ScenesAdCandidate
}

type ApplyScenesResultHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	outbox     port.OutboxRepository
}

func NewApplyScenesResultHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	outbox port.OutboxRepository,
) *ApplyScenesResultHandler {
	return &ApplyScenesResultHandler{
		media:      media,
		detections: detections,
		outbox:     outbox,
	}
}

func (h *ApplyScenesResultHandler) Handle(ctx context.Context, cmd ApplyScenesResultCommand) error {
	if cmd.ScenesKey == "" {
		return errors.New("scenesKey is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepScenes)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}
	// Ignore stale results from a previous attempt after reanalyze.
	if step != nil && cmd.Attempt > 0 && step.Attempt > cmd.Attempt {
		return nil
	}

	detector := cmd.Detector
	if detector == "" {
		detector = "scenes.pyscenedetect@1"
	}
	now := time.Now().UTC()

	const minConfidence = 0.55
	items := make([]*domaindetection.Detection, 0, len(cmd.AdCandidates))
	for _, c := range cmd.AdCandidates {
		if c.SourceEndMs <= c.SourceStartMs {
			continue
		}
		if c.Confidence != nil && *c.Confidence < minConfidence {
			continue
		}
		payload, _ := json.Marshal(map[string]any{
			"reason":          c.Reason,
			"cutCount":        c.CutCount,
			"scenesKey":       cmd.ScenesKey,
			"adCandidatesKey": cmd.AdCandidatesKey,
		})
		d := domaindetection.New(m.ID, domaindetection.KindAd, c.SourceStartMs, c.SourceEndMs, detector, payload)
		d.Confidence = c.Confidence
		items = append(items, d)
	}

	scenesMeta, _ := json.Marshal(map[string]any{
		"sceneCount":      cmd.SceneCount,
		"detector":        detector,
		"adCandidatesKey": cmd.AdCandidatesKey,
	})
	adMeta, _ := json.Marshal(map[string]any{
		"count":     len(items),
		"detector":  detector,
		"scenesKey": cmd.ScenesKey,
	})

	m.RecordDetectionsUpdated(string(domaindetection.KindAd))

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.SaveAsset(txCtx, domainmedia.NewAsset(m.ID, domainmedia.AssetKindScenes, cmd.ScenesKey, nil, scenesMeta)); err != nil {
			return err
		}
		if cmd.AdCandidatesKey != "" {
			if err := h.media.SaveAsset(txCtx, domainmedia.NewAsset(m.ID, domainmedia.AssetKindAdCandidates, cmd.AdCandidatesKey, nil, adMeta)); err != nil {
				return err
			}
		}
		if err := h.detections.DeleteByMediaAndKind(txCtx, m.ID, domaindetection.KindAd); err != nil {
			return err
		}
		if err := h.detections.SaveMany(txCtx, items); err != nil {
			return err
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := mediacmd.CompleteStep(txCtx, h.media, m.ID, domainmedia.StepScenes, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return errors.New("failed to apply scenes result")
	}
	return nil
}

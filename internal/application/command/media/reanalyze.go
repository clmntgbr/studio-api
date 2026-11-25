package media

import (
	"context"
	"errors"
	"time"

	domaindetection "go-api/internal/domain/detection"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ReanalyzeCommand struct {
	MediaID uuid.UUID
	Steps   []string // empty = silence only when threshold-related
}

type ReanalyzeHandler struct {
	media      domainmedia.WriteRepository
	detections domaindetection.WriteRepository
	outbox     port.OutboxRepository
	jobs       port.JobPublisher
}

func NewReanalyzeHandler(
	media domainmedia.WriteRepository,
	detections domaindetection.WriteRepository,
	outbox port.OutboxRepository,
	jobs port.JobPublisher,
) *ReanalyzeHandler {
	return &ReanalyzeHandler{
		media:      media,
		detections: detections,
		outbox:     outbox,
		jobs:       jobs,
	}
}

func (h *ReanalyzeHandler) Handle(ctx context.Context, cmd ReanalyzeCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	steps := cmd.Steps
	if len(steps) == 0 {
		steps = []string{string(domainmedia.StepSilence)}
	}

	for _, step := range steps {
		switch step {
		case string(domainmedia.StepSilence):
			if err := h.requeueSilence(ctx, m); err != nil {
				return err
			}
		case string(domainmedia.StepText):
			if err := h.requeueText(ctx, m); err != nil {
				return err
			}
		case string(domainmedia.StepLLM):
			if err := h.requeueLLM(ctx, m); err != nil {
				return err
			}
		case string(domainmedia.StepScenes):
			if err := h.requeueScenes(ctx, m); err != nil {
				return err
			}
		case string(domainmedia.StepReframe):
			if err := h.requeueReframe(ctx, m); err != nil {
				return err
			}
		default:
			return errors.New("unsupported reanalyze step")
		}
	}
	return nil
}

func (h *ReanalyzeHandler) requeueSilence(ctx context.Context, m *domainmedia.MediaFile) error {
	cfg, err := h.media.GetConfiguration(ctx, m.ID)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(m.ID)
	}
	if !cfg.SilenceEnabled {
		// Clear silence cuts, invalidate in-flight jobs, rebuild immediately.
		return h.clearDetectionsAndCancelStep(
			ctx,
			m.ID,
			string(domaindetection.KindSilence),
			domainmedia.StepSilence,
			[]domaindetection.Kind{domaindetection.KindSilence},
		)
	}

	asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindAnalysis)
	if err != nil {
		return err
	}
	if asset == nil || asset.StorageKey == "" {
		return errors.New("analysis audio not ready")
	}

	now := time.Now().UTC()
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepSilence)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(m.ID, domainmedia.StepSilence)
		step.Status = domainmedia.StepStatusQueued
		step.QueuedAt = &now
		step.Attempt = 1
		if err := h.media.SaveStep(ctx, step); err != nil {
			return err
		}
	} else {
		step.Status = domainmedia.StepStatusQueued
		step.Progress = 0
		step.ErrorMessage = nil
		step.StartedAt = nil
		step.CompletedAt = nil
		step.QueuedAt = &now
		step.Attempt++
		if step.Attempt < 1 {
			step.Attempt = 1
		}
		if err := h.media.UpdateStep(ctx, step); err != nil {
			return err
		}
	}

	return h.jobs.PublishSilenceJob(ctx, m.ID.String(), asset.StorageKey, port.SilenceJobOptions{
		ThresholdMode:      cfg.ThresholdMode,
		SilenceThresholdDb: cfg.SilenceThresholdDB,
		NoiseFloorOffsetDb: cfg.NoiseFloorOffsetDb(),
		Attempt:            step.Attempt,
	})
}

func (h *ReanalyzeHandler) requeueText(ctx context.Context, m *domainmedia.MediaFile) error {
	cfg, err := h.media.GetConfiguration(ctx, m.ID)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(m.ID)
	}

	var clearKinds []domaindetection.Kind
	if !cfg.FillersEnabled {
		clearKinds = append(clearKinds, domaindetection.KindFiller)
	}
	if !cfg.RepetitionsEnabled {
		clearKinds = append(clearKinds, domaindetection.KindRepetition)
	}
	if !cfg.FalseStartsEnabled {
		clearKinds = append(clearKinds, domaindetection.KindFalseStart)
	}

	anyEnabled := cfg.FillersEnabled || cfg.RepetitionsEnabled || cfg.FalseStartsEnabled
	if !anyEnabled {
		return h.clearDetectionsAndCancelStep(
			ctx,
			m.ID,
			"text",
			domainmedia.StepText,
			[]domaindetection.Kind{
				domaindetection.KindFiller,
				domaindetection.KindRepetition,
				domaindetection.KindFalseStart,
			},
		)
	}

	// Sync-delete kinds that were just turned off, then requeue for remaining.
	if len(clearKinds) > 0 {
		if err := h.clearDetectionsAndCancelStep(ctx, m.ID, "text", domainmedia.StepText, clearKinds); err != nil {
			return err
		}
		// clearDetectionsAndCancelStep cancels the step — re-queue below.
	}

	asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindWords)
	if err != nil {
		return err
	}
	if asset == nil || asset.StorageKey == "" {
		// Words not ready: disabled kinds already cleared; skip enqueue.
		return nil
	}

	now := time.Now().UTC()
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepText)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(m.ID, domainmedia.StepText)
		step.Status = domainmedia.StepStatusQueued
		step.QueuedAt = &now
		step.Attempt = 1
		if err := h.media.SaveStep(ctx, step); err != nil {
			return err
		}
	} else {
		step.Status = domainmedia.StepStatusQueued
		step.Progress = 0
		step.ErrorMessage = nil
		step.StartedAt = nil
		step.CompletedAt = nil
		step.QueuedAt = &now
		step.Attempt++
		if step.Attempt < 1 {
			step.Attempt = 1
		}
		if err := h.media.UpdateStep(ctx, step); err != nil {
			return err
		}
	}

	lang := ""
	if cfg.Language != nil {
		lang = *cfg.Language
	}
	return h.jobs.PublishTextJob(ctx, m.ID.String(), asset.StorageKey, port.TextJobOptions{
		Language:           lang,
		DetectionLevel:     cfg.DetectionLevel,
		FillersEnabled:     cfg.FillersEnabled,
		RepetitionsEnabled: cfg.RepetitionsEnabled,
		FalseStartsEnabled: cfg.FalseStartsEnabled,
		Attempt:            step.Attempt,
	})
}

func (h *ReanalyzeHandler) clearDetectionsAndCancelStep(
	ctx context.Context,
	mediaID uuid.UUID,
	eventKind string,
	stepKind domainmedia.StepKind,
	clearKinds []domaindetection.Kind,
) error {
	m, err := h.media.GetByID(ctx, mediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}
	m.RecordDetectionsUpdated(eventKind)
	now := time.Now().UTC()
	return h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		for _, k := range clearKinds {
			if err := h.detections.DeleteByMediaAndKind(txCtx, m.ID, k); err != nil {
				return err
			}
		}
		step, err := h.media.GetStep(txCtx, m.ID, stepKind)
		if err != nil {
			return err
		}
		if step == nil {
			step = domainmedia.NewProcessingStep(m.ID, stepKind)
			step.Status = domainmedia.StepStatusCancelled
			step.Attempt = 1
			step.Progress = 1
			step.CompletedAt = &now
			step.QueuedAt = &now
			if err := h.media.SaveStep(txCtx, step); err != nil {
				return err
			}
		} else {
			step.Status = domainmedia.StepStatusCancelled
			step.Progress = 1
			step.ErrorMessage = nil
			step.CompletedAt = &now
			step.Attempt++
			if step.Attempt < 1 {
				step.Attempt = 1
			}
			if err := h.media.UpdateStep(txCtx, step); err != nil {
				return err
			}
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
}

func (h *ReanalyzeHandler) requeueLLM(ctx context.Context, m *domainmedia.MediaFile) error {
	asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindWords)
	if err != nil {
		return err
	}
	if asset == nil || asset.StorageKey == "" {
		return errors.New("words not ready")
	}
	cfg, err := h.media.GetConfiguration(ctx, m.ID)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(m.ID)
	}

	now := time.Now().UTC()
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepLLM)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(m.ID, domainmedia.StepLLM)
		step.Status = domainmedia.StepStatusQueued
		step.QueuedAt = &now
		step.Attempt = 1
		if err := h.media.SaveStep(ctx, step); err != nil {
			return err
		}
	} else {
		step.Status = domainmedia.StepStatusQueued
		step.Progress = 0
		step.ErrorMessage = nil
		step.StartedAt = nil
		step.CompletedAt = nil
		step.QueuedAt = &now
		step.Attempt++
		if step.Attempt < 1 {
			step.Attempt = 1
		}
		if err := h.media.UpdateStep(ctx, step); err != nil {
			return err
		}
	}

	lang := ""
	if cfg.Language != nil {
		lang = *cfg.Language
	}
	return h.jobs.PublishLLMJob(ctx, m.ID.String(), asset.StorageKey, port.LLMJobOptions{
		Language:       lang,
		Tasks:          []string{"sponsors", "chapters", "viral"},
		SponsorsAction: string(cfg.SponsorsAction),
	})
}

func (h *ReanalyzeHandler) requeueScenes(ctx context.Context, m *domainmedia.MediaFile) error {
	asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindProxy)
	if err != nil {
		return err
	}
	if asset == nil || asset.StorageKey == "" {
		return errors.New("proxy not ready")
	}

	now := time.Now().UTC()
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepScenes)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(m.ID, domainmedia.StepScenes)
		step.Status = domainmedia.StepStatusQueued
		step.QueuedAt = &now
		step.Attempt = 1
		if err := h.media.SaveStep(ctx, step); err != nil {
			return err
		}
	} else {
		step.Status = domainmedia.StepStatusQueued
		step.Progress = 0
		step.ErrorMessage = nil
		step.StartedAt = nil
		step.CompletedAt = nil
		step.QueuedAt = &now
		step.Attempt++
		if step.Attempt < 1 {
			step.Attempt = 1
		}
		if err := h.media.UpdateStep(ctx, step); err != nil {
			return err
		}
	}

	return h.jobs.PublishScenesJob(ctx, m.ID.String(), asset.StorageKey, step.Attempt)
}

func (h *ReanalyzeHandler) requeueReframe(ctx context.Context, m *domainmedia.MediaFile) error {
	asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindProxy)
	if err != nil {
		return err
	}
	if asset == nil || asset.StorageKey == "" {
		return errors.New("proxy not ready")
	}

	now := time.Now().UTC()
	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepReframe)
	if err != nil {
		return err
	}
	if step == nil {
		step = domainmedia.NewProcessingStep(m.ID, domainmedia.StepReframe)
		step.Status = domainmedia.StepStatusQueued
		step.QueuedAt = &now
		step.Attempt = 1
		if err := h.media.SaveStep(ctx, step); err != nil {
			return err
		}
	} else {
		step.Status = domainmedia.StepStatusQueued
		step.Progress = 0
		step.ErrorMessage = nil
		step.StartedAt = nil
		step.CompletedAt = nil
		step.QueuedAt = &now
		step.Attempt++
		if step.Attempt < 1 {
			step.Attempt = 1
		}
		if err := h.media.UpdateStep(ctx, step); err != nil {
			return err
		}
	}

	return h.jobs.PublishReframeJob(ctx, m.ID.String(), asset.StorageKey, 0, 0)
}

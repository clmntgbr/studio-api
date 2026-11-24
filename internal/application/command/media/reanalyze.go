package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ReanalyzeCommand struct {
	MediaID uuid.UUID
	Steps   []string // empty = silence only when threshold-related
}

type ReanalyzeHandler struct {
	media  domainmedia.WriteRepository
	jobs   port.JobPublisher
}

func NewReanalyzeHandler(media domainmedia.WriteRepository, jobs port.JobPublisher) *ReanalyzeHandler {
	return &ReanalyzeHandler{media: media, jobs: jobs}
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
	asset, err := h.media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindAnalysis)
	if err != nil {
		return err
	}
	if asset == nil || asset.StorageKey == "" {
		return errors.New("analysis audio not ready")
	}
	cfg, err := h.media.GetConfiguration(ctx, m.ID)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(m.ID)
	}
	if !cfg.SilenceEnabled {
		return nil
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
	})
}

func (h *ReanalyzeHandler) requeueText(ctx context.Context, m *domainmedia.MediaFile) error {
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
	if !cfg.FillersEnabled && !cfg.RepetitionsEnabled && !cfg.FalseStartsEnabled {
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

	return h.jobs.PublishReframeJob(ctx, m.ID.String(), asset.StorageKey)
}

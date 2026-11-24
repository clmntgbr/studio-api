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
		step.CompletedAt = nil
		step.QueuedAt = &now
		step.Attempt++
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

package clip

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type AnalyzeCommand struct {
	ProjectID uuid.UUID
	MediaID   uuid.UUID
}

type AnalyzeHandler struct {
	media domainmedia.WriteRepository
	jobs  port.JobPublisher
}

func NewAnalyzeHandler(media domainmedia.WriteRepository, jobs port.JobPublisher) *AnalyzeHandler {
	return &AnalyzeHandler{media: media, jobs: jobs}
}

func (h *AnalyzeHandler) Handle(ctx context.Context, cmd AnalyzeCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil || m.ProjectID != cmd.ProjectID {
		return errors.New("media not found")
	}
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
	lang := ""
	if cfg.Language != nil {
		lang = *cfg.Language
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

	return h.jobs.PublishLLMJob(ctx, m.ID.String(), asset.StorageKey, port.LLMJobOptions{
		Language:       lang,
		Tasks:          []string{"sponsors", "chapters", "viral"},
		SponsorsAction: string(cfg.SponsorsAction),
	})
}

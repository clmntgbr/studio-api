package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type UpdateConfigurationCommand struct {
	MediaID              uuid.UUID
	SilenceEnabled       *bool
	ThresholdMode        *string
	SilenceThresholdDB   *float64
	DetectionLevel       *string
	SilenceMinDurationMs *int
	PaddingBeforeMs      *int
	PaddingAfterMs       *int
	SpeechMinDurationMs  *int
	FillersEnabled       *bool
	RepetitionsEnabled   *bool
	FalseStartsEnabled   *bool
	SponsorsAction       *string
	AdsAction            *string
	Language             *string
}

type UpdateConfigurationHandler struct {
	media domainmedia.WriteRepository
}

func NewUpdateConfigurationHandler(media domainmedia.WriteRepository) *UpdateConfigurationHandler {
	return &UpdateConfigurationHandler{media: media}
}

func (h *UpdateConfigurationHandler) Handle(ctx context.Context, cmd UpdateConfigurationCommand) (*domainmedia.Configuration, error) {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("media not found")
	}
	cfg, err := h.media.GetConfiguration(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = domainmedia.DefaultConfiguration(cmd.MediaID)
		if err := h.media.SaveConfiguration(ctx, cfg); err != nil {
			return nil, errors.New("failed to create configuration")
		}
	}

	if cmd.SilenceEnabled != nil {
		cfg.SilenceEnabled = *cmd.SilenceEnabled
	}
	if cmd.ThresholdMode != nil {
		mode := *cmd.ThresholdMode
		if mode != "auto" && mode != "manual" {
			return nil, errors.New("thresholdMode must be auto or manual")
		}
		cfg.ThresholdMode = mode
	}
	if cmd.SilenceThresholdDB != nil {
		cfg.SilenceThresholdDB = cmd.SilenceThresholdDB
	}
	if cmd.DetectionLevel != nil {
		level := *cmd.DetectionLevel
		switch level {
		case "gentle", "standard", "aggressive":
			cfg.DetectionLevel = level
		default:
			return nil, errors.New("detectionLevel must be gentle, standard or aggressive")
		}
	}
	if cmd.SilenceMinDurationMs != nil {
		cfg.SilenceMinDurationMs = *cmd.SilenceMinDurationMs
	}
	if cmd.PaddingBeforeMs != nil {
		cfg.PaddingBeforeMs = *cmd.PaddingBeforeMs
	}
	if cmd.PaddingAfterMs != nil {
		cfg.PaddingAfterMs = *cmd.PaddingAfterMs
	}
	if cmd.SpeechMinDurationMs != nil {
		cfg.SpeechMinDurationMs = *cmd.SpeechMinDurationMs
	}
	if cmd.FillersEnabled != nil {
		cfg.FillersEnabled = *cmd.FillersEnabled
	}
	if cmd.RepetitionsEnabled != nil {
		cfg.RepetitionsEnabled = *cmd.RepetitionsEnabled
	}
	if cmd.FalseStartsEnabled != nil {
		cfg.FalseStartsEnabled = *cmd.FalseStartsEnabled
	}
	if cmd.SponsorsAction != nil {
		cfg.SponsorsAction = domainmedia.DecisionAction(*cmd.SponsorsAction)
	}
	if cmd.AdsAction != nil {
		cfg.AdsAction = domainmedia.DecisionAction(*cmd.AdsAction)
	}
	if cmd.Language != nil {
		cfg.Language = cmd.Language
	}
	cfg.UpdatedAt = time.Now().UTC()

	if err := h.media.UpdateConfiguration(ctx, cfg); err != nil {
		return nil, errors.New("failed to update configuration")
	}
	return cfg, nil
}

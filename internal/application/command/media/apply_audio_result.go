package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type ApplyAudioResultCommand struct {
	MediaID          uuid.UUID
	AnalysisKey      string
	TranscriptionKey string
	MasterKey        string
}

type ApplyAudioResultHandler struct {
	media  domainmedia.WriteRepository
	outbox port.OutboxRepository
}

func NewApplyAudioResultHandler(media domainmedia.WriteRepository, outbox port.OutboxRepository) *ApplyAudioResultHandler {
	return &ApplyAudioResultHandler{media: media, outbox: outbox}
}

func (h *ApplyAudioResultHandler) Handle(ctx context.Context, cmd ApplyAudioResultCommand) error {
	if cmd.AnalysisKey == "" {
		return errors.New("analysisKey is required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepAudio)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	m.RecordAudioReady(cmd.AnalysisKey, cmd.TranscriptionKey, cmd.MasterKey)
	now := time.Now().UTC()

	return h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		assets := []*domainmedia.Asset{
			domainmedia.NewAsset(m.ID, domainmedia.AssetKindAnalysis, cmd.AnalysisKey, nil, nil),
			domainmedia.NewAsset(m.ID, domainmedia.AssetKindTranscription, cmd.TranscriptionKey, nil, nil),
			domainmedia.NewAsset(m.ID, domainmedia.AssetKindMaster, cmd.MasterKey, nil, nil),
		}
		for _, a := range assets {
			if a.StorageKey == "" {
				continue
			}
			if err := h.media.SaveAsset(txCtx, a); err != nil {
				return err
			}
		}
		if err := completeStep(txCtx, h.media, m.ID, domainmedia.StepAudio, now); err != nil {
			return err
		}
		if err := queueStep(txCtx, h.media, m.ID, domainmedia.StepWaveform, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
}

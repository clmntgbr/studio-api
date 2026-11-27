package transcript

import (
	"context"
	"errors"
	"time"

	mediacmd "go-api/internal/application/command/media"
	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"
	domaintranscript "go-api/internal/domain/transcript"

	"github.com/google/uuid"
)

type WordInput struct {
	Text          string
	SourceStartMs int64
	SourceEndMs   int64
	Confidence    *float64
	Speaker       *string
}

type ApplyResultCommand struct {
	MediaID       uuid.UUID
	WordsKey      string
	Language      string
	Text          string
	Provider      string
	ProviderJobID string
	Words         []WordInput
}

type ApplyResultHandler struct {
	media      domainmedia.WriteRepository
	transcript domaintranscript.WriteRepository
	outbox     port.OutboxRepository
}

func NewApplyResultHandler(
	media domainmedia.WriteRepository,
	transcript domaintranscript.WriteRepository,
	outbox port.OutboxRepository,
) *ApplyResultHandler {
	return &ApplyResultHandler{media: media, transcript: transcript, outbox: outbox}
}

func (h *ApplyResultHandler) Handle(ctx context.Context, cmd ApplyResultCommand) error {
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("media not found")
	}

	step, err := h.media.GetStep(ctx, m.ID, domainmedia.StepTranscribe)
	if err != nil {
		return err
	}
	if step != nil && step.Status == domainmedia.StepStatusCompleted {
		return nil
	}

	provider := cmd.Provider
	if provider == "" {
		provider = "assemblyai"
	}
	now := time.Now().UTC()
	lang := cmd.Language
	text := cmd.Text
	jobID := cmd.ProviderJobID

	tr := domaintranscript.New(m.ID)
	tr.Provider = provider
	tr.Status = domaintranscript.StatusCompleted
	if lang != "" {
		tr.Language = &lang
	}
	if text != "" {
		tr.Text = &text
	}
	if jobID != "" {
		tr.ProviderJobID = &jobID
	}

	words := make([]domaintranscript.Word, 0, len(cmd.Words))
	for i, w := range cmd.Words {
		if w.Text == "" || w.SourceEndMs < w.SourceStartMs {
			continue
		}
		words = append(words, domaintranscript.Word{
			TranscriptID:  tr.ID,
			WordIndex:     i,
			Text:          w.Text,
			SourceStartMs: w.SourceStartMs,
			SourceEndMs:   w.SourceEndMs,
			Confidence:    w.Confidence,
			Speaker:       w.Speaker,
		})
	}

	m.RecordTranscribed(cmd.WordsKey, lang)
	asset := domainmedia.NewAsset(m.ID, domainmedia.AssetKindWords, cmd.WordsKey, nil, nil)

	err = h.media.WithTransaction(ctx, func(txCtx context.Context) error {
		if err := h.transcript.Save(txCtx, tr); err != nil {
			return err
		}
		if err := h.transcript.ReplaceWords(txCtx, tr.ID, words); err != nil {
			return err
		}
		if err := h.media.SaveAsset(txCtx, asset); err != nil {
			return err
		}
		if err := h.media.Update(txCtx, m); err != nil {
			return err
		}
		if err := mediacmd.CompleteStep(txCtx, h.media, m.ID, domainmedia.StepTranscribe, now); err != nil {
			return err
		}
		return h.outbox.StoreEvents(txCtx, m.PullEvents())
	})
	if err != nil {
		return errors.New("failed to apply transcript result")
	}
	return mediacmd.TryMarkMediaReady(ctx, h.media, h.outbox, cmd.MediaID)
}

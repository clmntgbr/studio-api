package transcript

import (
	"context"
	"errors"

	domaintranscript "go-api/internal/domain/transcript"

	"github.com/google/uuid"
)

type GetTranscriptQuery struct {
	MediaID uuid.UUID
	FromMs  *int64
	ToMs    *int64
}

type TranscriptView struct {
	Transcript *domaintranscript.Transcript
	Words      []domaintranscript.Word
}

type GetTranscriptHandler struct {
	repo domaintranscript.ReadRepository
}

func NewGetTranscriptHandler(repo domaintranscript.ReadRepository) *GetTranscriptHandler {
	return &GetTranscriptHandler{repo: repo}
}

func (h *GetTranscriptHandler) Handle(ctx context.Context, q GetTranscriptQuery) (*TranscriptView, error) {
	tr, err := h.repo.FindByMediaID(ctx, q.MediaID)
	if err != nil {
		return nil, err
	}
	if tr == nil {
		return nil, errors.New("transcript not found")
	}
	words, err := h.repo.ListWords(ctx, tr.ID)
	if err != nil {
		return nil, err
	}
	out := make([]domaintranscript.Word, 0, len(words))
	for _, w := range words {
		if q.FromMs != nil && w.SourceEndMs < *q.FromMs {
			continue
		}
		if q.ToMs != nil && w.SourceStartMs > *q.ToMs {
			continue
		}
		out = append(out, w)
	}
	return &TranscriptView{Transcript: tr, Words: out}, nil
}

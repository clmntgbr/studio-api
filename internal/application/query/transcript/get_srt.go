package transcript

import (
	"context"
	"errors"

	domaintranscript "go-api/internal/domain/transcript"
	srtexport "go-api/internal/transcript"

	"github.com/google/uuid"
)

type GetSRTQuery struct {
	MediaID uuid.UUID
}

type GetSRTHandler struct {
	repo domaintranscript.ReadRepository
}

func NewGetSRTHandler(repo domaintranscript.ReadRepository) *GetSRTHandler {
	return &GetSRTHandler{repo: repo}
}

func (h *GetSRTHandler) Handle(ctx context.Context, q GetSRTQuery) (string, error) {
	tr, err := h.repo.FindByMediaID(ctx, q.MediaID)
	if err != nil {
		return "", err
	}
	if tr == nil {
		return "", errors.New("transcript not found")
	}
	words, err := h.repo.ListWords(ctx, tr.ID)
	if err != nil {
		return "", err
	}
	items := make([]srtexport.Word, 0, len(words))
	for _, w := range words {
		items = append(items, srtexport.Word{
			Text:          w.Text,
			SourceStartMs: w.SourceStartMs,
			SourceEndMs:   w.SourceEndMs,
		})
	}
	return srtexport.ToSRT(items), nil
}

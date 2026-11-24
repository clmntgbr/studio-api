package presenter

import (
	"time"

	querytranscript "go-api/internal/application/query/transcript"
)

type TranscriptWordResponse struct {
	WordIndex     int      `json:"wordIndex"`
	Text          string   `json:"text"`
	SourceStartMs int64    `json:"sourceStartMs"`
	SourceEndMs   int64    `json:"sourceEndMs"`
	Confidence    *float64 `json:"confidence,omitempty"`
	Speaker       *string  `json:"speaker,omitempty"`
}

type TranscriptResponse struct {
	ID            string                   `json:"id"`
	MediaFileID   string                   `json:"mediaFileId"`
	Language      *string                  `json:"language,omitempty"`
	Text          *string                  `json:"text,omitempty"`
	Provider      string                   `json:"provider"`
	ProviderJobID *string                  `json:"providerJobId,omitempty"`
	Status        string                   `json:"status"`
	CreatedAt     time.Time                `json:"createdAt"`
	Words         []TranscriptWordResponse `json:"words"`
}

func NewTranscriptResponse(v *querytranscript.TranscriptView) TranscriptResponse {
	tr := v.Transcript
	words := make([]TranscriptWordResponse, 0, len(v.Words))
	for _, w := range v.Words {
		words = append(words, TranscriptWordResponse{
			WordIndex:     w.WordIndex,
			Text:          w.Text,
			SourceStartMs: w.SourceStartMs,
			SourceEndMs:   w.SourceEndMs,
			Confidence:    w.Confidence,
			Speaker:       w.Speaker,
		})
	}
	return TranscriptResponse{
		ID:            tr.ID.String(),
		MediaFileID:   tr.MediaFileID.String(),
		Language:      tr.Language,
		Text:          tr.Text,
		Provider:      tr.Provider,
		ProviderJobID: tr.ProviderJobID,
		Status:        string(tr.Status),
		CreatedAt:     tr.CreatedAt,
		Words:         words,
	}
}

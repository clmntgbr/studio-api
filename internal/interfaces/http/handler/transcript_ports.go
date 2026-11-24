package handler

import (
	"context"

	querytranscript "go-api/internal/application/query/transcript"
)

type transcriptGetHandler interface {
	Handle(ctx context.Context, q querytranscript.GetTranscriptQuery) (*querytranscript.TranscriptView, error)
}

type transcriptGetSRTHandler interface {
	Handle(ctx context.Context, q querytranscript.GetSRTQuery) (string, error)
}

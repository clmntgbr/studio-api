package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type StreamQuery struct {
	MediaID     uuid.UUID
	Kind        string
	RangeHeader string
}

type StreamResult struct {
	Stream *port.ObjectStream
	Kind   string
}

type GetStreamHandler struct {
	media   domainmedia.WriteRepository
	storage port.ObjectStorage
	bucket  string
}

func NewGetStreamHandler(
	media domainmedia.WriteRepository,
	storage port.ObjectStorage,
	bucket string,
) *GetStreamHandler {
	return &GetStreamHandler{media: media, storage: storage, bucket: bucket}
}

func (h *GetStreamHandler) Handle(ctx context.Context, q StreamQuery) (*StreamResult, error) {
	if q.MediaID == uuid.Nil {
		return nil, errors.New("mediaId is required")
	}
	key, kind, err := resolvePlaybackObject(ctx, h.media, q.MediaID, q.Kind)
	if err != nil {
		return nil, err
	}

	stream, err := h.storage.GetObjectStream(ctx, h.bucket, key, q.RangeHeader)
	if err != nil {
		return nil, errors.New("failed to open media object")
	}
	return &StreamResult{Stream: stream, Kind: kind}, nil
}

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
	kind := q.Kind
	if kind == "" {
		kind = domainmedia.AssetKindProxy
	}
	if kind != domainmedia.AssetKindProxy {
		return nil, errors.New("kind must be proxy")
	}

	m, err := h.media.GetByID(ctx, q.MediaID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errors.New("media not found")
	}

	asset, err := h.media.GetAssetByKind(ctx, m.ID, kind)
	if err != nil {
		return nil, err
	}
	if asset == nil || asset.StorageKey == "" {
		return nil, errors.New("proxy not ready")
	}

	stream, err := h.storage.GetObjectStream(ctx, h.bucket, asset.StorageKey, q.RangeHeader)
	if err != nil {
		return nil, errors.New("failed to open proxy object")
	}
	return &StreamResult{Stream: stream}, nil
}

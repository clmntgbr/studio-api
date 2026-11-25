package media

import (
	"context"
	"errors"
	"time"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type PlaybackQuery struct {
	MediaID uuid.UUID
	Kind    string // proxy (default)
}

type PlaybackResult struct {
	URL       string
	Kind      string
	ExpiresIn int
}

type GetPlaybackHandler struct {
	media      domainmedia.WriteRepository
	storage    port.ObjectStorage
	bucket     string
	publicBase string
	expiry     time.Duration
}

func NewGetPlaybackHandler(
	media domainmedia.WriteRepository,
	storage port.ObjectStorage,
	bucket, publicBase string,
) *GetPlaybackHandler {
	return &GetPlaybackHandler{
		media:      media,
		storage:    storage,
		bucket:     bucket,
		publicBase: publicBase,
		expiry:     60 * time.Minute,
	}
}

func (h *GetPlaybackHandler) Handle(ctx context.Context, q PlaybackQuery) (*PlaybackResult, error) {
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
	url, err := h.storage.PresignGetObject(ctx, h.bucket, asset.StorageKey, h.expiry, h.publicBase)
	if err != nil {
		return nil, errors.New("failed to presign playback")
	}
	return &PlaybackResult{
		URL:       url,
		Kind:      kind,
		ExpiresIn: int(h.expiry.Seconds()),
	}, nil
}

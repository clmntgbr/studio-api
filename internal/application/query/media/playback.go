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
	Kind    string // proxy | source | auto (default)
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
	key, kind, err := resolvePlaybackObject(ctx, h.media, q.MediaID, q.Kind)
	if err != nil {
		return nil, err
	}
	url, err := h.storage.PresignGetObject(ctx, h.bucket, key, h.expiry, h.publicBase)
	if err != nil {
		return nil, errors.New("failed to presign playback")
	}
	return &PlaybackResult{
		URL:       url,
		Kind:      kind,
		ExpiresIn: int(h.expiry.Seconds()),
	}, nil
}

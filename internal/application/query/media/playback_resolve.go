package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

const (
	playbackKindProxy  = domainmedia.AssetKindProxy
	playbackKindSource = "source"
	playbackKindAuto   = "auto"
)

func normalizePlaybackKind(kind string) string {
	switch kind {
	case "", playbackKindAuto:
		return playbackKindAuto
	case playbackKindProxy, playbackKindSource:
		return kind
	default:
		return ""
	}
}

// resolvePlaybackObject picks the MinIO key for playback.
// auto: proxy if present, else original source.
func resolvePlaybackObject(
	ctx context.Context,
	media domainmedia.WriteRepository,
	mediaID uuid.UUID,
	kind string,
) (objectKey string, resolvedKind string, err error) {
	norm := normalizePlaybackKind(kind)
	if norm == "" {
		return "", "", errors.New("kind must be proxy, source, or auto")
	}

	m, err := media.GetByID(ctx, mediaID)
	if err != nil {
		return "", "", err
	}
	if m == nil {
		return "", "", errors.New("media not found")
	}

	tryProxy := func() (string, bool, error) {
		asset, err := media.GetAssetByKind(ctx, m.ID, domainmedia.AssetKindProxy)
		if err != nil {
			return "", false, err
		}
		if asset == nil || asset.StorageKey == "" {
			return "", false, nil
		}
		return asset.StorageKey, true, nil
	}
	trySource := func() (string, bool) {
		if m.StorageKey == nil || *m.StorageKey == "" {
			return "", false
		}
		return *m.StorageKey, true
	}

	switch norm {
	case playbackKindProxy:
		key, ok, err := tryProxy()
		if err != nil {
			return "", "", err
		}
		if !ok {
			return "", "", errors.New("proxy not ready")
		}
		return key, playbackKindProxy, nil
	case playbackKindSource:
		key, ok := trySource()
		if !ok {
			return "", "", errors.New("source not ready")
		}
		return key, playbackKindSource, nil
	default: // auto
		if key, ok, err := tryProxy(); err != nil {
			return "", "", err
		} else if ok {
			return key, playbackKindProxy, nil
		}
		if key, ok := trySource(); ok {
			return key, playbackKindSource, nil
		}
		return "", "", errors.New("playback not ready")
	}
}

package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"

	"github.com/google/uuid"
)

type GetConfigurationQuery struct {
	MediaID uuid.UUID
}

type GetConfigurationHandler struct {
	media domainmedia.ReadRepository
}

func NewGetConfigurationHandler(media domainmedia.ReadRepository) *GetConfigurationHandler {
	return &GetConfigurationHandler{media: media}
}

func (h *GetConfigurationHandler) Handle(ctx context.Context, q GetConfigurationQuery) (*domainmedia.Configuration, error) {
	cfg, err := h.media.GetConfiguration(ctx, q.MediaID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, errors.New("configuration not found")
	}
	return cfg, nil
}

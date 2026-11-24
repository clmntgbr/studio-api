package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type PresignPartsCommand struct {
	MediaID     uuid.UUID
	UploadID    string
	PartNumbers []int32
}

type PresignPartsHandler struct {
	media   domainmedia.WriteRepository
	storage port.ObjectStorage
	bucket  string
	public  string
}

func NewPresignPartsHandler(
	media domainmedia.WriteRepository,
	storage port.ObjectStorage,
	bucket, publicEndpoint string,
) *PresignPartsHandler {
	return &PresignPartsHandler{media: media, storage: storage, bucket: bucket, public: publicEndpoint}
}

func (h *PresignPartsHandler) Handle(ctx context.Context, cmd PresignPartsCommand) ([]port.UploadPartURL, error) {
	if cmd.UploadID == "" || len(cmd.PartNumbers) == 0 {
		return nil, errors.New("uploadId and partNumbers are required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.StorageKey == nil {
		return nil, errors.New("media not found")
	}
	if m.Status != domainmedia.StatusPending {
		return nil, errors.New("media is not awaiting upload")
	}
	return h.storage.PresignUploadParts(ctx, h.bucket, *m.StorageKey, cmd.UploadID, cmd.PartNumbers, 0, h.public)
}

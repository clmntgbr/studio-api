package media

import (
	"context"
	"errors"

	domainmedia "go-api/internal/domain/media"
	"go-api/internal/domain/port"

	"github.com/google/uuid"
)

type CompleteUploadCommand struct {
	MediaID  uuid.UUID
	UploadID string
	Parts    []port.CompletedUploadPart
}

// CompleteUploadHandler assembles multipart parts in MinIO.
// Pipeline start is MarkSourceReady (also triggered by the MinIO object-created webhook).
type CompleteUploadHandler struct {
	media     domainmedia.WriteRepository
	storage   port.ObjectStorage
	bucket    string
	markReady *MarkSourceReadyHandler
}

func NewCompleteUploadHandler(
	media domainmedia.WriteRepository,
	storage port.ObjectStorage,
	bucket string,
	markReady *MarkSourceReadyHandler,
) *CompleteUploadHandler {
	return &CompleteUploadHandler{media: media, storage: storage, bucket: bucket, markReady: markReady}
}

func (h *CompleteUploadHandler) Handle(ctx context.Context, cmd CompleteUploadCommand) (*domainmedia.MediaFile, error) {
	if cmd.UploadID == "" || len(cmd.Parts) == 0 {
		return nil, errors.New("uploadId and parts are required")
	}
	m, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if m == nil || m.StorageKey == nil {
		return nil, errors.New("media not found")
	}
	if m.Status != domainmedia.StatusPending && m.Status != domainmedia.StatusProcessing {
		return nil, errors.New("media is not awaiting upload")
	}

	// Idempotent re-complete: object may already exist after a previous success.
	if m.Status == domainmedia.StatusPending {
		if err := h.storage.CompleteMultipartUpload(ctx, h.bucket, *m.StorageKey, cmd.UploadID, cmd.Parts); err != nil {
			return nil, errors.New("failed to complete multipart upload")
		}
	}

	var sizePtr *int64
	if size, err := h.storage.HeadObjectSize(ctx, h.bucket, *m.StorageKey); err == nil && size > 0 {
		sizePtr = &size
	}

	// Shared post-source path (same as MinIO webhook / fetch result).
	if err := h.markReady.Handle(ctx, MarkSourceReadyCommand{
		MediaID:   m.ID,
		SourceKey: *m.StorageKey,
		SizeBytes: sizePtr,
	}); err != nil {
		return nil, err
	}

	updated, err := h.media.GetByID(ctx, cmd.MediaID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, errors.New("media not found")
	}
	return updated, nil
}

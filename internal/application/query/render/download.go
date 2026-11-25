package render

import (
	"context"
	"errors"
	"fmt"

	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type DownloadQuery struct {
	ID          uuid.UUID
	RangeHeader string
}

type DownloadResult struct {
	Stream   *port.ObjectStream
	Filename string
}

type DownloadHandler struct {
	repo         domainrender.ReadRepository
	storage      port.ObjectStorage
	renderBucket string
}

func NewDownloadHandler(
	repo domainrender.ReadRepository,
	storage port.ObjectStorage,
	renderBucket string,
) *DownloadHandler {
	return &DownloadHandler{
		repo:         repo,
		storage:      storage,
		renderBucket: renderBucket,
	}
}

func (h *DownloadHandler) Handle(ctx context.Context, q DownloadQuery) (*DownloadResult, error) {
	r, err := h.repo.FindByID(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, errors.New("render not found")
	}
	if r.Status != domainrender.StatusCompleted {
		return nil, errors.New("render is not ready")
	}
	if r.StorageKey == nil || *r.StorageKey == "" {
		return nil, errors.New("render output missing")
	}
	stream, err := h.storage.GetObjectStream(ctx, h.renderBucket, *r.StorageKey, q.RangeHeader)
	if err != nil {
		return nil, errors.New("failed to open render object")
	}
	if stream.ContentType == "" {
		stream.ContentType = "video/mp4"
	}
	return &DownloadResult{
		Stream:   stream,
		Filename: fmt.Sprintf("render-%s.mp4", r.ID.String()),
	}, nil
}

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
	Part        int // 0 = full master; 1..N = outputParts[part-1]
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
	key := ""
	filename := fmt.Sprintf("render-%s.mp4", r.ID.String())
	if q.Part > 0 {
		if q.Part > len(r.OutputParts) {
			return nil, errors.New("render part not found")
		}
		key = r.OutputParts[q.Part-1]
		filename = fmt.Sprintf("render-%s-part-%02d.mp4", r.ID.String(), q.Part)
	} else if r.StorageKey != nil {
		key = *r.StorageKey
	}
	if key == "" {
		return nil, errors.New("render output missing")
	}
	stream, err := h.storage.GetObjectStream(ctx, h.renderBucket, key, q.RangeHeader)
	if err != nil {
		return nil, errors.New("failed to open render object")
	}
	if stream.ContentType == "" {
		stream.ContentType = "video/mp4"
	}
	return &DownloadResult{
		Stream:   stream,
		Filename: filename,
	}, nil
}

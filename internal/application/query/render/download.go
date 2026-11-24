package render

import (
	"context"
	"errors"
	"time"

	"go-api/internal/domain/port"
	domainrender "go-api/internal/domain/render"

	"github.com/google/uuid"
)

type DownloadQuery struct {
	ID uuid.UUID
}

type DownloadResult struct {
	URL       string
	ExpiresIn int
}

type DownloadHandler struct {
	repo         domainrender.ReadRepository
	storage      port.ObjectStorage
	renderBucket string
	publicBase   string
	expiry       time.Duration
}

func NewDownloadHandler(
	repo domainrender.ReadRepository,
	storage port.ObjectStorage,
	renderBucket, publicBase string,
) *DownloadHandler {
	return &DownloadHandler{
		repo:         repo,
		storage:      storage,
		renderBucket: renderBucket,
		publicBase:   publicBase,
		expiry:       15 * time.Minute,
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
	url, err := h.storage.PresignGetObject(ctx, h.renderBucket, *r.StorageKey, h.expiry, h.publicBase)
	if err != nil {
		return nil, errors.New("failed to presign download")
	}
	return &DownloadResult{
		URL:       url,
		ExpiresIn: int(h.expiry.Seconds()),
	}, nil
}

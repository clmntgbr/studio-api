package detection

import (
	"context"

	domaindetection "go-api/internal/domain/detection"

	"github.com/google/uuid"
)

type ListDetectionsQuery struct {
	MediaID uuid.UUID
	Kind    string
	FromMs  *int64
	ToMs    *int64
}

type ListDetectionsHandler struct {
	repo domaindetection.ReadRepository
}

func NewListDetectionsHandler(repo domaindetection.ReadRepository) *ListDetectionsHandler {
	return &ListDetectionsHandler{repo: repo}
}

func (h *ListDetectionsHandler) Handle(ctx context.Context, q ListDetectionsQuery) ([]domaindetection.Detection, error) {
	items, err := h.repo.ListByMedia(ctx, q.MediaID)
	if err != nil {
		return nil, err
	}
	out := make([]domaindetection.Detection, 0, len(items))
	for _, d := range items {
		if q.Kind != "" && string(d.Kind) != q.Kind {
			continue
		}
		if q.FromMs != nil && d.SourceEndMs < *q.FromMs {
			continue
		}
		if q.ToMs != nil && d.SourceStartMs > *q.ToMs {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

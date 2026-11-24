package detection

import (
	"context"

	domaindetection "go-api/internal/domain/detection"

	"github.com/google/uuid"
)

type ListDetectionsQuery struct {
	MediaID uuid.UUID
	Kind    string
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
	if q.Kind == "" {
		return items, nil
	}
	out := make([]domaindetection.Detection, 0, len(items))
	for _, d := range items {
		if string(d.Kind) == q.Kind {
			out = append(out, d)
		}
	}
	return out, nil
}

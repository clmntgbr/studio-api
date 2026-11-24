package llmartifact

import (
	"context"

	domainllm "go-api/internal/domain/llmartifact"

	"github.com/google/uuid"
)

type ListQuery struct {
	MediaID uuid.UUID
	Kind    string
}

type ListHandler struct {
	repo domainllm.ReadRepository
}

func NewListHandler(repo domainllm.ReadRepository) *ListHandler {
	return &ListHandler{repo: repo}
}

func (h *ListHandler) Handle(ctx context.Context, q ListQuery) ([]domainllm.Artifact, error) {
	if q.Kind != "" {
		return h.repo.ListByMediaAndKind(ctx, q.MediaID, q.Kind)
	}
	return h.repo.ListByMedia(ctx, q.MediaID)
}

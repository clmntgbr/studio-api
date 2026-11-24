package llmartifact

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Save(ctx context.Context, a *Artifact) error
}

type ReadRepository interface {
	ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]Artifact, error)
}

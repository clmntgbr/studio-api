package llmartifact

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Save(ctx context.Context, a *Artifact) error
	DeleteByMediaAndKind(ctx context.Context, mediaFileID uuid.UUID, kind string) error
}

type ReadRepository interface {
	ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]Artifact, error)
	ListByMediaAndKind(ctx context.Context, mediaFileID uuid.UUID, kind string) ([]Artifact, error)
}

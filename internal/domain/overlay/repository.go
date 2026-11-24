package overlay

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	Save(ctx context.Context, o *Overlay) error
	Update(ctx context.Context, o *Overlay) error
}

type ReadRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]Overlay, error)
}

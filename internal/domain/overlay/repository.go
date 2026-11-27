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
	ListByClip(ctx context.Context, clipID uuid.UUID) ([]Overlay, error)
}

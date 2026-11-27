package clip

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, c *Clip) error
	Update(ctx context.Context, c *Clip) error
	DeleteByMediaAndOrigin(ctx context.Context, mediaFileID uuid.UUID, origin string) error
}

type ReadRepository interface {
	ListByMedia(ctx context.Context, mediaFileID uuid.UUID) ([]Clip, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Clip, error)
}

package clip

import (
	"context"

	"github.com/google/uuid"
)

type WriteRepository interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	Save(ctx context.Context, c *Clip) error
	Update(ctx context.Context, c *Clip) error
}

type ReadRepository interface {
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]Clip, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Clip, error)
}
